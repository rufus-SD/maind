package store

import (
	"fmt"
	"time"

	"github.com/rufus-SD/maind/internal/crypto"
	"github.com/rufus-SD/maind/internal/model"

	"github.com/google/uuid"
)

func (s *Store) CreateScan(scan *model.Scan) error {
	if scan.ID == "" {
		scan.ID = uuid.New().String()
	}
	scan.StartedAt = time.Now().UTC()
	scan.Status = "running"

	thoughts := scan.Thoughts
	thoughtsEnc := false
	if s.key != nil && thoughts != "" {
		enc, err := crypto.Encrypt([]byte(thoughts), s.key)
		if err == nil {
			thoughts = enc
			thoughtsEnc = true
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO scans (id, project, source, status, thoughts, thoughts_encrypted, started_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		scan.ID, scan.Project, scan.Source, scan.Status, thoughts, thoughtsEnc,
		scan.StartedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) AppendScanThought(scanID, thought string) error {
	var existing string
	var encrypted int
	err := s.db.QueryRow("SELECT thoughts, thoughts_encrypted FROM scans WHERE id = ?", scanID).Scan(&existing, &encrypted)
	if err != nil {
		short := scanID
		if len(short) > 8 {
			short = short[:8]
		}
		return fmt.Errorf("scan %s not found", short)
	}

	if encrypted == 1 && s.key != nil {
		dec, err := crypto.Decrypt(existing, s.key)
		if err == nil {
			existing = string(dec)
		}
	}

	if existing != "" {
		existing += "\n"
	}
	existing += fmt.Sprintf("[%s] %s", time.Now().UTC().Format("15:04:05"), thought)

	thoughts := existing
	thoughtsEnc := false
	if s.key != nil {
		enc, err := crypto.Encrypt([]byte(thoughts), s.key)
		if err == nil {
			thoughts = enc
			thoughtsEnc = true
		}
	}

	_, err = s.db.Exec("UPDATE scans SET thoughts = ?, thoughts_encrypted = ? WHERE id = ?",
		thoughts, thoughtsEnc, scanID)
	return err
}

func (s *Store) CompleteScan(scanID, summary string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM entries WHERE scan_id = ?", scanID).Scan(&count)

	summaryText := summary
	summaryEnc := false
	if s.key != nil && summary != "" {
		enc, err := crypto.Encrypt([]byte(summary), s.key)
		if err == nil {
			summaryText = enc
			summaryEnc = true
		}
	}

	_, err := s.db.Exec(`UPDATE scans SET status = 'completed', summary = ?, summary_encrypted = ?,
		entries_created = ?, completed_at = ? WHERE id = ?`,
		summaryText, summaryEnc, count, now, scanID)
	return err
}

func (s *Store) FailScan(scanID, reason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec("UPDATE scans SET status = 'failed', summary = ?, completed_at = ? WHERE id = ?",
		reason, now, scanID)
	return err
}

func (s *Store) ResolveScanID(prefix string) (string, error) {
	if len(prefix) == 36 {
		return prefix, nil
	}
	rows, err := s.db.Query("SELECT id FROM scans WHERE id LIKE ?", prefix+"%")
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
		return "", fmt.Errorf("no scan matching %q", prefix)
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("ambiguous prefix %q matches %d scans", prefix, len(ids))
	}
}

func (s *Store) GetScan(scanID string) (*model.Scan, error) {
	resolvedID, err := s.ResolveScanID(scanID)
	if err != nil {
		return nil, err
	}

	var scan model.Scan
	var summary, completedAt *string
	var thoughtsEnc, summaryEnc int
	var startedAt string

	err = s.db.QueryRow(`SELECT id, project, source, status, summary, summary_encrypted,
		thoughts, thoughts_encrypted, entries_created, started_at, completed_at
		FROM scans WHERE id = ?`, resolvedID).Scan(
		&scan.ID, &scan.Project, &scan.Source, &scan.Status, &summary, &summaryEnc,
		&scan.Thoughts, &thoughtsEnc, &scan.EntriesCreated, &startedAt, &completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan not found")
	}

	scan.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	if completedAt != nil {
		t, _ := time.Parse(time.RFC3339, *completedAt)
		scan.CompletedAt = &t
	}
	if summary != nil {
		scan.Summary = *summary
	}

	if thoughtsEnc == 1 && s.key != nil {
		dec, err := crypto.Decrypt(scan.Thoughts, s.key)
		if err == nil {
			scan.Thoughts = string(dec)
			scan.ThoughtsEncrypted = false
		} else {
			scan.ThoughtsEncrypted = true
		}
	}
	if summaryEnc == 1 && s.key != nil && scan.Summary != "" {
		dec, err := crypto.Decrypt(scan.Summary, s.key)
		if err == nil {
			scan.Summary = string(dec)
			scan.SummaryEncrypted = false
		} else {
			scan.SummaryEncrypted = true
		}
	}

	return &scan, nil
}

func (s *Store) ListScans(limit int) ([]model.Scan, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(`SELECT id, project, source, status, summary, summary_encrypted,
		entries_created, started_at, completed_at
		FROM scans ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []model.Scan
	for rows.Next() {
		var scan model.Scan
		var summary, completedAt *string
		var summaryEnc int
		var startedAt string

		err := rows.Scan(&scan.ID, &scan.Project, &scan.Source, &scan.Status,
			&summary, &summaryEnc, &scan.EntriesCreated, &startedAt, &completedAt)
		if err != nil {
			return nil, err
		}

		scan.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		if completedAt != nil {
			t, _ := time.Parse(time.RFC3339, *completedAt)
			scan.CompletedAt = &t
		}
		if summary != nil {
			scan.Summary = *summary
			if summaryEnc == 1 && s.key != nil {
				dec, err := crypto.Decrypt(scan.Summary, s.key)
				if err == nil {
					scan.Summary = string(dec)
				}
			}
		}

		scans = append(scans, scan)
	}
	return scans, rows.Err()
}

func (s *Store) LinkEntryToScan(entryID, scanID string) error {
	_, err := s.db.Exec("UPDATE entries SET scan_id = ? WHERE id = ?", scanID, entryID)
	return err
}

func (s *Store) ScanEntries(scanID string) ([]model.Entry, error) {
	rows, err := s.db.Query(`SELECT e.id, e.kind, e.title, e.body, e.body_encrypted,
		e.importance, e.source, e.project, e.created_at, e.updated_at, e.deleted_at
		FROM entries e WHERE e.scan_id = ? ORDER BY e.created_at`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanEntries(rows)
}
