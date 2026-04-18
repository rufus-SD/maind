package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/rufus-SD/maind/internal/config"
	"github.com/rufus-SD/maind/internal/model"
	storemod "github.com/rufus-SD/maind/internal/store"
	"github.com/rufus-SD/maind/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Try Maind instantly with sample data",
	Long: `Launch a temporary brain populated with realistic memories.
No setup needed — see what Maind does in 10 seconds.

The demo data is deleted when you exit.`,
	RunE: runDemo,
}

type demoEntry struct {
	kind       model.EntryKind
	title      string
	body       string
	importance int
	project    string
	tags       []string
	source     string
	daysAgo    int
}

var demoEntries = []demoEntry{
	{model.KindDecision, "Auth strategy", "Use JWT with short-lived access tokens (15min) and refresh tokens (7d). Chose over server sessions because API is stateless and serves mobile + web clients. Rotation handled in middleware/auth.go.", 9, "myapp", []string{"auth", "jwt", "api"}, "ide", 12},
	{model.KindSolution, "OOM fix", "Fixed out-of-memory crash in /export endpoint. Root cause: loading entire dataset into memory. Switched to streaming JSON encoder that writes row-by-row. Memory usage dropped from 2GB to 40MB for large exports.", 8, "myapp", []string{"performance", "memory", "go"}, "ide", 10},
	{model.KindBug, "Migration rollback gap", "Migrations 002 and 003 have no rollback (down) path. If either fails in production, manual SQL intervention is required. Added to tech debt backlog.", 7, "myapp", []string{"database", "migration", "tech-debt"}, "ide", 8},
	{model.KindDecision, "Redis cache layer", "Added Redis for session cache and rate limiting. Not for application data — SQLite handles that. Redis is ephemeral; if it dies, app degrades gracefully (slower, no rate limit).", 8, "myapp", []string{"redis", "cache", "architecture"}, "ide", 7},
	{model.KindLearning, "Context timeout pattern", "Always wrap DB calls with context.WithTimeout. Discovered after a deadlocked query held a connection for 3 minutes and cascaded into pool exhaustion. 5s timeout is our standard.", 9, "myapp", []string{"go", "database", "patterns"}, "ide", 15},
	{model.KindSnippet, "Graceful shutdown", "Use errgroup with signal.NotifyContext for clean shutdown. Drain HTTP connections, flush logs, close DB — in that order. See cmd/server/main.go for reference implementation.", 6, "myapp", []string{"go", "patterns", "devops"}, "ide", 20},
	{model.KindDecision, "Monorepo structure", "Frontend (Next.js) and backend (Go) in one repo. Shared proto definitions in /api. Chose monorepo over polyrepo because team is small (3 devs) and deploys are coupled anyway.", 7, "myapp", []string{"architecture", "monorepo"}, "ide", 25},
	{model.KindContext, "Deployment pipeline", "GitHub Actions builds Docker image, pushes to GCR, deploys to Cloud Run. Staging auto-deploys on push to main. Production requires manual approval. No Terraform yet — infra is simple enough.", 6, "myapp", []string{"ci-cd", "deployment", "gcp"}, "ide", 18},
	{model.KindBug, "Timezone rendering", "All timestamps stored as UTC in the database. Frontend was rendering them as UTC too instead of converting to user's local timezone. Fixed in utils/time.ts — always use formatLocal().", 5, "myapp", []string{"frontend", "timezone", "bug"}, "ide", 5},
	{model.KindLearning, "SQLite WAL mode", "Enable WAL mode for SQLite in production. Default journal mode locks the entire database on writes. WAL allows concurrent reads during writes. One pragma: journal_mode=WAL.", 8, "", []string{"sqlite", "database", "performance"}, "cli", 30},
	{model.KindDecision, "Error handling convention", "All API errors return {code, message, details}. Internal errors get logged but never exposed. 5xx returns generic message + request ID for support debugging.", 6, "billing-api", []string{"api", "errors", "convention"}, "ide", 14},
	{model.KindSolution, "N+1 query fix", "Product listing was doing 1 query per product for category lookup. Replaced with a JOIN and batch prefetch. Page load went from 1200ms to 80ms on 500-product pages.", 9, "billing-api", []string{"sql", "performance", "optimization"}, "ide", 3},
	{model.KindLearning, "Go embed for migrations", "Use //go:embed to bundle SQL migration files into the binary. No need to ship a migrations/ directory or worry about file paths in production. Clean single-binary deployment.", 5, "", []string{"go", "deployment", "patterns"}, "cli", 22},
	{model.KindContext, "Team conventions", "PR reviews require 1 approval. Squash merge only. Branch naming: feat/, fix/, chore/. Commit messages follow conventional commits. No direct push to main.", 4, "", []string{"team", "git", "conventions"}, "cli", 40},
	{model.KindDecision, "No ORM", "Using raw SQL with database/sql instead of GORM or ent. Team is comfortable with SQL, queries are complex enough that ORM abstractions would leak anyway. sqlc for type-safe query generation.", 7, "billing-api", []string{"go", "database", "orm"}, "ide", 16},
}

func runDemo(cmd *cobra.Command, args []string) error {
	tmpDir, err := os.MkdirTemp("", "maind-demo-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Version:           1,
		Name:              "demo",
		EncryptionEnabled: false,
		DBPath:            "maind.db",
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	os.MkdirAll(tmpDir, 0755)
	if err := cfg.Save(tmpDir); err != nil {
		return fmt.Errorf("save demo config: %w", err)
	}

	s, err := storemod.New(cfg, tmpDir, nil)
	if err != nil {
		return fmt.Errorf("open demo store: %w", err)
	}
	if err := s.Migrate(); err != nil {
		s.Close()
		return fmt.Errorf("migrate: %w", err)
	}

	now := time.Now().UTC()
	for _, d := range demoEntries {
		entry := &model.Entry{
			Kind:       d.kind,
			Title:      d.title,
			Body:       d.body,
			Importance: d.importance,
			Project:    d.project,
			Source:     d.source,
			Tags:       d.tags,
		}
		if err := s.CreateEntry(entry); err != nil {
			s.Close()
			return fmt.Errorf("create demo entry: %w", err)
		}

		created := now.Add(-time.Duration(d.daysAgo) * 24 * time.Hour)
		s.DB().Exec("UPDATE entries SET created_at = ? WHERE id = ?",
			created.Format(time.RFC3339), entry.ID)

		s.LogActivity("STORE", fmt.Sprintf("%s: %s", d.kind, truncBody(d.body, 50)), entry.ID)
	}

	scanID := "demo-scan-001"
	s.DB().Exec(`INSERT INTO scans (id, project, source, status, thoughts, summary, entries_created, started_at, completed_at)
		VALUES (?, 'myapp', 'ide', 'completed', ?, ?, 5, ?, ?)`,
		scanID,
		"[10:25:23] Analyzing go.mod — 14 direct deps, none pinned. Risky for reproducible builds.\n[10:26:01] Found 3 SQL migrations with no rollback. Recovery from bad deploys is impossible.\n[10:27:15] Auth uses JWT with refresh tokens — stateless API, serves mobile + web.\n[10:28:02] Redis is ephemeral cache only — app degrades gracefully if Redis dies.\n[10:28:44] README mentions monitoring but no alerting config found anywhere.",
		"Analyzed myapp: found 4 architectural decisions, 2 tech debt items, 1 missing config",
		now.Add(-2*time.Hour).Format(time.RFC3339),
		now.Add(-2*time.Hour+3*time.Minute).Format(time.RFC3339),
	)
	s.LogActivity("SCAN_DONE", "entries=5 project=myapp", scanID)

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Welcome to Maind.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  This is a demo brain with 15 sample memories from real-world projects.")
	fmt.Fprintln(os.Stderr, "  Try these commands:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "    recall auth          — search your memories")
	fmt.Fprintln(os.Stderr, "    list decision        — list all decisions")
	fmt.Fprintln(os.Stderr, "    scan                 — see the AI scan that analyzed 'myapp'")
	fmt.Fprintln(os.Stderr, "    scan show demo       — see the AI's full thought log")
	fmt.Fprintln(os.Stderr, "    remember \"something\" — store a new memory")
	fmt.Fprintln(os.Stderr, "    help                 — all commands")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Everything here is temporary — it's deleted when you exit.")
	fmt.Fprintln(os.Stderr)

	m := tui.NewModel(s, "demo", false)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		s.Close()
		return err
	}

	s.Close()
	fmt.Fprintln(os.Stderr, "  Demo data cleaned up. Run 'maind init' to create your real brain.")
	return nil
}
