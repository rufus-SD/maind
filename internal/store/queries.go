package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rufus-SD/maind/internal/crypto"
	"github.com/rufus-SD/maind/internal/model"

	"github.com/google/uuid"
)

type ListOptions struct {
	Kind           string
	Tag            string
	Project        string
	Limit          int
	Offset         int
	SortBy         string
	SortOrder      string
	IncludeDeleted bool
}

type SearchOptions struct {
	Kind    string
	Tag     string
	Project string
	Limit   int
}

func (s *Store) CreateEntry(e *model.Entry) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	e.CreatedAt = now
	e.UpdatedAt = now

	body := e.Body
	encrypted := false
	if s.key != nil {
		enc, err := crypto.Encrypt([]byte(body), s.key)
		if err != nil {
			return fmt.Errorf("encrypt body: %w", err)
		}
		body = enc
		encrypted = true
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO entries (id, kind, title, body, body_encrypted, importance, source, project, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Kind, nullStr(e.Title), body, boolToInt(encrypted),
		e.Importance, e.Source, nullStr(e.Project),
		e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert entry: %w", err)
	}

	for _, tagName := range e.Tags {
		tag, err := s.getOrCreateTagTx(tx, tagName)
		if err != nil {
			return fmt.Errorf("create tag %q: %w", tagName, err)
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO entry_tags (entry_id, tag_id) VALUES (?, ?)", e.ID, tag.ID); err != nil {
			return fmt.Errorf("link tag: %w", err)
		}
	}

	ftsBody := ""
	if !encrypted {
		ftsBody = e.Body
	}
	if len(e.Tags) > 0 {
		ftsBody += " " + strings.Join(e.Tags, " ")
	}
	if _, err := tx.Exec("INSERT INTO entries_fts (entry_id, title, body) VALUES (?, ?, ?)", e.ID, e.Title, ftsBody); err != nil {
		return fmt.Errorf("insert fts: %w", err)
	}

	return tx.Commit()
}

func (s *Store) GetEntry(id string) (*model.Entry, error) {
	resolvedID, err := s.ResolveID(id)
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRow(`
		SELECT id, kind, title, body, body_encrypted, importance, source, project, created_at, updated_at, deleted_at
		FROM entries WHERE id = ?`, resolvedID)

	e, err := s.scanEntry(row)
	if err != nil {
		return nil, err
	}

	tags, err := s.getEntryTags(e.ID)
	if err != nil {
		return nil, err
	}
	e.Tags = tags
	return e, nil
}

func (s *Store) ListEntries(opts ListOptions) ([]model.Entry, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.SortBy == "" {
		opts.SortBy = "created_at"
	}
	if opts.SortOrder == "" {
		opts.SortOrder = "DESC"
	}

	validSort := map[string]bool{"created_at": true, "updated_at": true, "importance": true, "kind": true}
	if !validSort[opts.SortBy] {
		opts.SortBy = "created_at"
	}
	if opts.SortOrder != "ASC" && opts.SortOrder != "DESC" {
		opts.SortOrder = "DESC"
	}

	query := `SELECT e.id, e.kind, e.title, e.body, e.body_encrypted, e.importance, e.source, e.project, e.created_at, e.updated_at, e.deleted_at FROM entries e`
	var conditions []string
	var args []any

	if opts.Tag != "" {
		query += ` JOIN entry_tags et ON et.entry_id = e.id JOIN tags t ON t.id = et.tag_id`
		conditions = append(conditions, "t.name = ?")
		args = append(args, opts.Tag)
	}
	if !opts.IncludeDeleted {
		conditions = append(conditions, "e.deleted_at IS NULL")
	}
	if opts.Kind != "" {
		conditions = append(conditions, "e.kind = ?")
		args = append(args, opts.Kind)
	}
	if opts.Project != "" {
		conditions = append(conditions, "e.project = ?")
		args = append(args, opts.Project)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY e.%s %s LIMIT ? OFFSET ?", opts.SortBy, opts.SortOrder)
	args = append(args, opts.Limit, opts.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	return s.scanEntries(rows)
}

func (s *Store) SearchEntries(query string, opts SearchOptions) ([]model.Entry, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	ftsQuery := buildFTSQuery(query)

	sqlQuery := `
		SELECT e.id, e.kind, e.title, e.body, e.body_encrypted, e.importance, e.source, e.project, e.created_at, e.updated_at, e.deleted_at
		FROM entries_fts f
		JOIN entries e ON e.id = f.entry_id
		WHERE entries_fts MATCH ? AND e.deleted_at IS NULL`
	args := []any{ftsQuery}

	if opts.Kind != "" {
		sqlQuery += " AND e.kind = ?"
		args = append(args, opts.Kind)
	}
	if opts.Project != "" {
		sqlQuery += " AND e.project = ?"
		args = append(args, opts.Project)
	}

	sqlQuery += " ORDER BY rank LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return s.searchFallback(query, opts)
	}
	defer rows.Close()

	results, err := s.scanEntries(rows)
	if err != nil {
		return s.searchFallback(query, opts)
	}
	if len(results) == 0 {
		return s.searchFallback(query, opts)
	}
	return results, nil
}

func (s *Store) searchFallback(query string, opts SearchOptions) ([]model.Entry, error) {
	pattern := "%" + query + "%"
	sqlQuery := `
		SELECT DISTINCT e.id, e.kind, e.title, e.body, e.body_encrypted, e.importance, e.source, e.project, e.created_at, e.updated_at, e.deleted_at
		FROM entries e
		LEFT JOIN entry_tags et ON et.entry_id = e.id
		LEFT JOIN tags t ON t.id = et.tag_id
		WHERE e.deleted_at IS NULL AND (e.title LIKE ? OR e.body LIKE ? OR t.name LIKE ?)`
	args := []any{pattern, pattern, pattern}

	if opts.Kind != "" {
		sqlQuery += " AND e.kind = ?"
		args = append(args, opts.Kind)
	}
	if opts.Project != "" {
		sqlQuery += " AND e.project = ?"
		args = append(args, opts.Project)
	}

	sqlQuery += " ORDER BY e.importance DESC, e.created_at DESC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search fallback: %w", err)
	}
	defer rows.Close()
	return s.scanEntries(rows)
}

func (s *Store) SoftDeleteEntry(id string) error {
	resolvedID, err := s.ResolveID(id)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec("UPDATE entries SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", now, now, resolvedID)
	if err != nil {
		return fmt.Errorf("soft delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("entry %s not found or already archived", id)
	}
	return nil
}

func (s *Store) CreateLink(l *model.Link) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	l.CreatedAt = time.Now().UTC()
	if l.Weight == 0 {
		l.Weight = 1.0
	}
	if l.Metadata == "" {
		l.Metadata = "{}"
	}

	fromID, err := s.ResolveID(l.FromEntryID)
	if err != nil {
		return fmt.Errorf("resolve from-entry: %w", err)
	}
	toID, err := s.ResolveID(l.ToEntryID)
	if err != nil {
		return fmt.Errorf("resolve to-entry: %w", err)
	}
	l.FromEntryID = fromID
	l.ToEntryID = toID

	_, err = s.db.Exec(`
		INSERT INTO links (id, from_entry_id, to_entry_id, relation, weight, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.FromEntryID, l.ToEntryID, l.Relation, l.Weight, l.Metadata,
		l.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetEntryLinks(entryID string) ([]model.Link, error) {
	resolvedID, err := s.ResolveID(entryID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT id, from_entry_id, to_entry_id, relation, weight, metadata, created_at
		FROM links WHERE from_entry_id = ? OR to_entry_id = ?
		ORDER BY created_at DESC`, resolvedID, resolvedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.Link
	for rows.Next() {
		var l model.Link
		var createdAt string
		if err := rows.Scan(&l.ID, &l.FromEntryID, &l.ToEntryID, &l.Relation, &l.Weight, &l.Metadata, &createdAt); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		links = append(links, l)
	}
	return links, rows.Err()
}

func (s *Store) ListTags() ([]model.Tag, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.name, COUNT(et.entry_id) as cnt, t.created_at
		FROM tags t
		LEFT JOIN entry_tags et ON et.tag_id = t.id
		LEFT JOIN entries e ON e.id = et.entry_id AND e.deleted_at IS NULL
		GROUP BY t.id
		ORDER BY cnt DESC, t.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		var createdAt string
		if err := rows.Scan(&t.ID, &t.Name, &t.Count, &createdAt); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *Store) ExportAll() (*model.Export, error) {
	entries, err := s.ListEntries(ListOptions{Limit: 100000, IncludeDeleted: true})
	if err != nil {
		return nil, err
	}
	tags, err := s.ListTags()
	if err != nil {
		return nil, err
	}

	var links []model.Link
	rows, err := s.db.Query(`SELECT id, from_entry_id, to_entry_id, relation, weight, metadata, created_at FROM links ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var l model.Link
		var createdAt string
		if err := rows.Scan(&l.ID, &l.FromEntryID, &l.ToEntryID, &l.Relation, &l.Weight, &l.Metadata, &createdAt); err != nil {
			return nil, err
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		links = append(links, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.Export{
		Version:    1,
		Entries:    entries,
		Tags:       tags,
		Links:      links,
		ExportedAt: time.Now().UTC(),
	}, nil
}

func (s *Store) ResolveID(prefix string) (string, error) {
	if len(prefix) == 36 {
		return prefix, nil
	}

	rows, err := s.db.Query("SELECT id FROM entries WHERE id LIKE ?", prefix+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}

	switch len(ids) {
	case 0:
		return "", fmt.Errorf("no entry matching %q", prefix)
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("ambiguous prefix %q matches %d entries", prefix, len(ids))
	}
}

// --- helpers ---

func (s *Store) getOrCreateTagTx(tx *sql.Tx, name string) (*model.Tag, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	var t model.Tag
	var createdAt string
	err := tx.QueryRow("SELECT id, name, created_at FROM tags WHERE name = ?", name).Scan(&t.ID, &t.Name, &createdAt)
	if err == nil {
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		return &t, nil
	}

	t = model.Tag{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	_, err = tx.Exec("INSERT INTO tags (id, name, created_at) VALUES (?, ?, ?)",
		t.ID, t.Name, t.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) getEntryTags(entryID string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT t.name FROM tags t
		JOIN entry_tags et ON et.tag_id = t.id
		WHERE et.entry_id = ?
		ORDER BY t.name`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tags = append(tags, name)
	}
	return tags, rows.Err()
}

func (s *Store) scanEntry(row *sql.Row) (*model.Entry, error) {
	var e model.Entry
	var title, project, deletedAt sql.NullString
	var createdAt, updatedAt string
	var encrypted int

	err := row.Scan(&e.ID, &e.Kind, &title, &e.Body, &encrypted, &e.Importance, &e.Source, &project, &createdAt, &updatedAt, &deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entry not found")
		}
		return nil, err
	}

	e.Title = title.String
	e.Project = project.String
	e.BodyEncrypted = encrypted == 1
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if deletedAt.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAt.String)
		e.DeletedAt = &t
	}

	if e.BodyEncrypted && s.key != nil {
		decrypted, err := crypto.Decrypt(e.Body, s.key)
		if err == nil {
			e.Body = string(decrypted)
			e.BodyEncrypted = false
		}
	}

	return &e, nil
}

func (s *Store) scanEntries(rows *sql.Rows) ([]model.Entry, error) {
	var entries []model.Entry
	for rows.Next() {
		var e model.Entry
		var title, project, deletedAt sql.NullString
		var createdAt, updatedAt string
		var encrypted int

		err := rows.Scan(&e.ID, &e.Kind, &title, &e.Body, &encrypted, &e.Importance, &e.Source, &project, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, err
		}

		e.Title = title.String
		e.Project = project.String
		e.BodyEncrypted = encrypted == 1
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if deletedAt.Valid {
			t, _ := time.Parse(time.RFC3339, deletedAt.String)
			e.DeletedAt = &t
		}

		if e.BodyEncrypted && s.key != nil {
			decrypted, err := crypto.Decrypt(e.Body, s.key)
			if err == nil {
				e.Body = string(decrypted)
				e.BodyEncrypted = false
			}
		}

		tags, _ := s.getEntryTags(e.ID)
		e.Tags = tags
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func buildFTSQuery(input string) string {
	words := strings.Fields(input)
	if len(words) == 0 {
		return input
	}
	for i, w := range words {
		if !strings.ContainsAny(w, `"*:^`) {
			words[i] = w + "*"
		}
	}
	return strings.Join(words, " ")
}
