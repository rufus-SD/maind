package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rufus-SD/maind/internal/model"
	"github.com/rufus-SD/maind/internal/store"
)

func refreshContextFile(s *store.Store, projectDir string) error {
	currentProject := filepath.Base(projectDir)

	entries, err := s.ListEntries(store.ListOptions{
		Limit:     50,
		SortBy:    "importance",
		SortOrder: "DESC",
	})
	if err != nil {
		return err
	}

	var local, global []string
	encryptedCount := 0
	for _, e := range entries {
		if e.BodyEncrypted {
			encryptedCount++
			continue
		}
		line := formatContextEntry(e)
		if strings.EqualFold(e.Project, currentProject) {
			local = append(local, line)
		} else {
			global = append(global, line)
		}
	}

	var buf strings.Builder
	buf.WriteString("# Maind — Your Memory Context\n\n")
	buf.WriteString(fmt.Sprintf("_Last refreshed: %s · current project: %s_\n\n", time.Now().Local().Format("2006-01-02 15:04:05"), currentProject))

	if len(local) > 0 {
		buf.WriteString(fmt.Sprintf("## This project (%s)\n\n", currentProject))
		for _, l := range local {
			buf.WriteString(l)
		}
		buf.WriteString("\n")
	}

	if len(global) > 0 {
		buf.WriteString("## Other memories\n\n")
		cap := 30
		if cap > len(global) {
			cap = len(global)
		}
		for _, l := range global[:cap] {
			buf.WriteString(l)
		}
	}

	if len(local)+len(global) == 0 {
		if encryptedCount > 0 {
			buf.WriteString("Session locked — run 'maind' in a terminal to unlock and load your memories.\n")
		} else {
			buf.WriteString("No memories stored yet.\n")
		}
	}

	dir := filepath.Join(projectDir, ".maind")
	os.MkdirAll(dir, 0755)
	ensureMaindGitignore(dir)
	// 0600: context.md holds DECRYPTED memory snippets, so it must not be
	// world-readable even though the brain itself is encrypted at rest.
	return os.WriteFile(filepath.Join(dir, "context.md"), []byte(buf.String()), 0600)
}

// ensureMaindGitignore writes a .gitignore inside the project's .maind directory
// so decrypted context (and anything else maind drops there) is never committed.
func ensureMaindGitignore(maindDir string) {
	p := filepath.Join(maindDir, ".gitignore")
	if _, err := os.Stat(p); err == nil {
		return
	}
	os.WriteFile(p, []byte("# maind writes DECRYPTED memory context here — never commit it\n*\n"), 0600)
}

func formatContextEntry(e model.Entry) string {
	body := strings.ReplaceAll(e.Body, "\n", " ")
	if len(body) > 200 {
		body = body[:200] + "..."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("- **[%s]** `%s` (importance: %d)", shortID(e.ID), e.Kind, e.Importance))
	if e.Project != "" {
		sb.WriteString(fmt.Sprintf(" _%s_", e.Project))
	}
	if len(e.Tags) > 0 {
		sb.WriteString(fmt.Sprintf(" [%s]", strings.Join(e.Tags, ", ")))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n", body))
	return sb.String()
}
