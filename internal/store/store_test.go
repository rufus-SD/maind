package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rufus-SD/maind/internal/config"
	"github.com/rufus-SD/maind/internal/crypto"
	"github.com/rufus-SD/maind/internal/model"
)

func newTestStore(t *testing.T, key []byte) *Store {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		Version:           1,
		Name:              "test",
		EncryptionEnabled: key != nil,
		DBPath:            "test.db",
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	s, err := New(cfg, dir, key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testKey() []byte {
	salt, _ := crypto.GenerateSalt()
	return crypto.DeriveKey("test-pass", salt)
}

func TestMigrate(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Version: 1, DBPath: "test.db"}
	s, err := New(cfg, dir, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate (idempotent): %v", err)
	}

	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if count != 3 {
		t.Errorf("migration count = %d, want 3", count)
	}
}

func TestCreateAndGetEntry(t *testing.T) {
	s := newTestStore(t, nil)

	entry := &model.Entry{
		Kind:       model.KindDecision,
		Body:       "Use PostgreSQL for prod",
		Tags:       []string{"database", "architecture"},
		Importance: 8,
		Source:     "cli",
		Project:    "myapp",
	}
	if err := s.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("entry ID not set after create")
	}

	got, err := s.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if got.Body != "Use PostgreSQL for prod" {
		t.Errorf("body = %q, want %q", got.Body, "Use PostgreSQL for prod")
	}
	if got.Kind != model.KindDecision {
		t.Errorf("kind = %q, want %q", got.Kind, model.KindDecision)
	}
	if got.Importance != 8 {
		t.Errorf("importance = %d, want 8", got.Importance)
	}
	if len(got.Tags) != 2 {
		t.Errorf("tags count = %d, want 2", len(got.Tags))
	}
}

func TestCreateEntryWithEncryption(t *testing.T) {
	key := testKey()
	s := newTestStore(t, key)

	entry := &model.Entry{
		Kind:       model.KindNote,
		Body:       "secret note",
		Importance: 5,
		Source:     "cli",
	}
	if err := s.CreateEntry(entry); err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}

	got, err := s.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if got.Body != "secret note" {
		t.Errorf("body = %q, want %q (should be decrypted)", got.Body, "secret note")
	}
	if got.BodyEncrypted {
		t.Error("BodyEncrypted = true after decryption, want false")
	}
}

func TestEncryptedEntryWrongKey(t *testing.T) {
	key1 := testKey()
	key2 := testKey()

	dir := t.TempDir()
	cfg := &config.Config{Version: 1, EncryptionEnabled: true, DBPath: "test.db"}

	s1, _ := New(cfg, dir, key1)
	s1.Migrate()
	entry := &model.Entry{Kind: model.KindNote, Body: "secret", Importance: 5, Source: "cli"}
	s1.CreateEntry(entry)
	s1.Close()

	s2, _ := New(cfg, dir, key2)
	s2.Migrate()
	defer s2.Close()

	got, err := s2.GetEntry(entry.ID)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if got.Body == "secret" {
		t.Error("decrypted with wrong key — should remain encrypted")
	}
	if !got.BodyEncrypted {
		t.Error("BodyEncrypted = false, want true (wrong key)")
	}
}

func TestListEntries(t *testing.T) {
	s := newTestStore(t, nil)

	for i, body := range []string{"first", "second", "third"} {
		s.CreateEntry(&model.Entry{
			Kind: model.KindNote, Body: body, Importance: i + 1, Source: "cli",
		})
	}

	entries, err := s.ListEntries(ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("count = %d, want 3", len(entries))
	}
}

func TestListEntriesFilterByKind(t *testing.T) {
	s := newTestStore(t, nil)

	s.CreateEntry(&model.Entry{Kind: model.KindDecision, Body: "d1", Importance: 5, Source: "cli"})
	s.CreateEntry(&model.Entry{Kind: model.KindBug, Body: "b1", Importance: 5, Source: "cli"})
	s.CreateEntry(&model.Entry{Kind: model.KindDecision, Body: "d2", Importance: 5, Source: "cli"})

	entries, err := s.ListEntries(ListOptions{Kind: "decision", Limit: 10})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("count = %d, want 2", len(entries))
	}
}

func TestSearchEntries(t *testing.T) {
	s := newTestStore(t, nil)

	s.CreateEntry(&model.Entry{Kind: model.KindDecision, Body: "use JWT for authentication", Importance: 7, Source: "cli"})
	s.CreateEntry(&model.Entry{Kind: model.KindNote, Body: "grocery list for tomorrow", Importance: 2, Source: "cli"})

	results, err := s.SearchEntries("JWT authentication", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("SearchEntries: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("search returned 0 results, want >= 1")
	}
	if results[0].Body != "use JWT for authentication" {
		t.Errorf("first result = %q, want JWT entry", results[0].Body)
	}
}

func TestSoftDelete(t *testing.T) {
	s := newTestStore(t, nil)

	entry := &model.Entry{Kind: model.KindNote, Body: "to delete", Importance: 1, Source: "cli"}
	s.CreateEntry(entry)

	if err := s.SoftDeleteEntry(entry.ID); err != nil {
		t.Fatalf("SoftDeleteEntry: %v", err)
	}

	entries, _ := s.ListEntries(ListOptions{Limit: 10})
	if len(entries) != 0 {
		t.Errorf("list after delete = %d, want 0", len(entries))
	}

	entries, _ = s.ListEntries(ListOptions{Limit: 10, IncludeDeleted: true})
	if len(entries) != 1 {
		t.Errorf("list with deleted = %d, want 1", len(entries))
	}
}

func TestActivityLog(t *testing.T) {
	s := newTestStore(t, nil)

	s.LogActivity("TEST", "test action", "")
	s.LogActivity("TEST", "another action", "")

	activities, err := s.RecentActivity(10)
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("activity count = %d, want 2", len(activities))
	}
}

func TestDatabaseFileCreated(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{Version: 1, DBPath: "test.db"}
	s, err := New(cfg, dir, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Migrate()
	s.Close()

	dbPath := filepath.Join(dir, "test.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("database file not created at %s", dbPath)
	}
}
