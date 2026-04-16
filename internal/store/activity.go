package store

import "time"

type Activity struct {
	ID        int
	Action    string
	Summary   string
	EntryID   string
	CreatedAt time.Time
}

type Stats struct {
	Entries  int
	Archived int
	Tags     int
	Links    int
}

func (s *Store) LogActivity(action, summary, entryID string) {
	s.db.Exec(
		"INSERT INTO activity_log (action, summary, entry_id, created_at) VALUES (?, ?, ?, ?)",
		action, summary, entryID, time.Now().UTC().Format(time.RFC3339),
	)
}

func (s *Store) RecentActivity(limit int) ([]Activity, error) {
	rows, err := s.db.Query(
		"SELECT id, action, summary, COALESCE(entry_id, ''), created_at FROM activity_log ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var a Activity
		var createdAt string
		if err := rows.Scan(&a.ID, &a.Action, &a.Summary, &a.EntryID, &createdAt); err != nil {
			continue
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		activities = append(activities, a)
	}
	return activities, rows.Err()
}

func (s *Store) Stats() Stats {
	var st Stats
	s.db.QueryRow("SELECT COUNT(*) FROM entries WHERE deleted_at IS NULL").Scan(&st.Entries)
	s.db.QueryRow("SELECT COUNT(*) FROM entries WHERE deleted_at IS NOT NULL").Scan(&st.Archived)
	s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&st.Tags)
	s.db.QueryRow("SELECT COUNT(*) FROM links").Scan(&st.Links)
	return st
}
