package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rufus-SD/maind/internal/store"
)

func refreshContextFile(s *store.Store, projectDir string) error {
	entries, err := s.ListEntries(store.ListOptions{
		Limit:     30,
		SortBy:    "importance",
		SortOrder: "DESC",
	})
	if err != nil {
		return err
	}

	var buf strings.Builder
	buf.WriteString("# Maind — Your Memory Context\n\n")
	buf.WriteString(fmt.Sprintf("_Last refreshed: %s_\n\n", time.Now().Local().Format("2006-01-02 15:04:05")))

	readable := 0
	for _, e := range entries {
		if e.BodyEncrypted {
			continue
		}
		readable++

		body := strings.ReplaceAll(e.Body, "\n", " ")
		if len(body) > 200 {
			body = body[:200] + "..."
		}

		buf.WriteString(fmt.Sprintf("- **[%s]** `%s` (importance: %d)", shortID(e.ID), e.Kind, e.Importance))
		if e.Project != "" {
			buf.WriteString(fmt.Sprintf(" _project: %s_", e.Project))
		}
		if len(e.Tags) > 0 {
			buf.WriteString(fmt.Sprintf(" [%s]", strings.Join(e.Tags, ", ")))
		}
		buf.WriteString("\n")
		buf.WriteString(fmt.Sprintf("  %s\n", body))
	}

	if readable == 0 {
		buf.WriteString("No readable memories. Is the session unlocked?\n")
	}

	dir := filepath.Join(projectDir, ".maind")
	os.MkdirAll(dir, 0755)
	return os.WriteFile(filepath.Join(dir, "context.md"), []byte(buf.String()), 0644)
}
