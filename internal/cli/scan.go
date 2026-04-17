package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/rufus-SD/maind/internal/model"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Track AI-driven knowledge scans",
	Long: `Manage scan sessions where an AI tool analyzes a project and populates
your memory. Each scan tracks what the AI looked at, what it thought,
and what memories it created.

Subcommands:
  start      Begin a new scan session
  log        Append a thought/observation to a running scan
  complete   Mark a scan as done with a summary
  list       Show recent scans
  show       Display full scan details`,
}

var scanStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Begin a new scan session",
	Long: `Start a tracked scan session. Returns a scan ID that the AI tool uses
to link its thoughts and created memories to this session.

Examples:
  maind scan start --project myapp --source ide
  maind scan start --project myapp --thought "Analyzing git history for architectural decisions"`,
	RunE: runScanStart,
}

var scanLogCmd = &cobra.Command{
	Use:   "log [scan-id] [thought]",
	Short: "Append a thought to a running scan",
	Long: `Record what the AI is observing or deciding during a scan.
These thoughts are encrypted and become the audit trail.

Examples:
  maind scan log abc123 "Found 3 migration files with no rollback strategy"
  maind scan log abc123 "README mentions Redis cache but no config found"`,
	Args: cobra.ExactArgs(2),
	RunE: runScanLog,
}

var scanCompleteCmd = &cobra.Command{
	Use:   "complete [scan-id]",
	Short: "Mark a scan as done",
	Long: `Finalize a scan session with an optional summary.
Automatically counts how many memories were created during this scan.

Examples:
  maind scan complete abc123 --summary "Extracted 8 architectural decisions and 3 known bugs"
  maind scan complete abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runScanComplete,
}

var scanListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show recent scans",
	Long: `Display recent scan sessions with their status, project, and entry count.

Examples:
  maind scan list
  maind scan list --limit 5`,
	RunE: runScanList,
}

var scanShowCmd = &cobra.Command{
	Use:   "show [scan-id]",
	Short: "Display full scan details",
	Long: `Show a scan's complete details including the AI's thought log,
summary, and all memories created during the session.

Examples:
  maind scan show abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runScanShow,
}

var (
	flagScanProject string
	flagScanSource  string
	flagScanThought string
	flagScanSummary string
	flagScanLimit   int
)

func init() {
	scanStartCmd.Flags().StringVarP(&flagScanProject, "project", "p", "", "project being scanned")
	scanStartCmd.Flags().StringVar(&flagScanSource, "source", "ide", "who initiated (cli|ide|api)")
	scanStartCmd.Flags().StringVar(&flagScanThought, "thought", "", "initial thought/observation")

	scanCompleteCmd.Flags().StringVar(&flagScanSummary, "summary", "", "scan summary")

	scanListCmd.Flags().IntVar(&flagScanLimit, "limit", 10, "max scans to show")

	scanCmd.AddCommand(scanStartCmd)
	scanCmd.AddCommand(scanLogCmd)
	scanCmd.AddCommand(scanCompleteCmd)
	scanCmd.AddCommand(scanListCmd)
	scanCmd.AddCommand(scanShowCmd)
}

func runScanStart(cmd *cobra.Command, args []string) error {
	if flagScanProject == "" {
		return fmt.Errorf("--project is required")
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	scan := &model.Scan{
		Project:  flagScanProject,
		Source:   flagScanSource,
		Thoughts: flagScanThought,
	}
	if err := s.CreateScan(scan); err != nil {
		return fmt.Errorf("create scan: %w", err)
	}

	s.LogActivity("SCAN_START", fmt.Sprintf("project=%s source=%s", scan.Project, scan.Source), scan.ID)

	fmt.Fprintf(os.Stderr, "Scan started [%s]\n", shortID(scan.ID))
	fmt.Fprintf(os.Stderr, "  Project: %s | Source: %s\n", scan.Project, scan.Source)
	fmt.Println(scan.ID)
	return nil
}

func runScanLog(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	scanID := args[0]
	thought := args[1]

	resolved, err := s.ResolveScanID(scanID)
	if err != nil {
		return err
	}

	if err := s.AppendScanThought(resolved, thought); err != nil {
		return fmt.Errorf("log thought: %w", err)
	}

	s.LogActivity("SCAN_LOG", truncBody(thought, 60), resolved)
	fmt.Fprintf(os.Stderr, "Logged to scan [%s]\n", shortID(resolved))
	return nil
}

func runScanComplete(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	scanID := args[0]
	resolved, err := s.ResolveScanID(scanID)
	if err != nil {
		return err
	}

	if err := s.CompleteScan(resolved, flagScanSummary); err != nil {
		return fmt.Errorf("complete scan: %w", err)
	}

	scan, _ := s.GetScan(resolved)
	s.LogActivity("SCAN_DONE", fmt.Sprintf("entries=%d project=%s", scan.EntriesCreated, scan.Project), resolved)

	fmt.Fprintf(os.Stderr, "Scan completed [%s]\n", shortID(resolved))
	fmt.Fprintf(os.Stderr, "  Entries created: %d\n", scan.EntriesCreated)
	if scan.Summary != "" {
		fmt.Fprintf(os.Stderr, "  Summary: %s\n", scan.Summary)
	}
	return nil
}

func runScanList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	scans, err := s.ListScans(flagScanLimit)
	if err != nil {
		return fmt.Errorf("list scans: %w", err)
	}

	if len(scans) == 0 {
		fmt.Fprintln(os.Stderr, "No scans yet.")
		return nil
	}

	for _, sc := range scans {
		status := sc.Status
		switch status {
		case "running":
			status = "RUNNING"
		case "completed":
			status = "DONE"
		case "failed":
			status = "FAILED"
		}

		date := sc.StartedAt.Format("2006-01-02 15:04")
		duration := ""
		if sc.CompletedAt != nil {
			d := sc.CompletedAt.Sub(sc.StartedAt)
			if d.Minutes() < 1 {
				duration = fmt.Sprintf(" (%ds)", int(d.Seconds()))
			} else {
				duration = fmt.Sprintf(" (%dm)", int(d.Minutes()))
			}
		}

		fmt.Printf("[%s] %-8s %s  %-20s  entries:%d%s\n",
			shortID(sc.ID), status, date, sc.Project, sc.EntriesCreated, duration)

		if sc.Summary != "" {
			fmt.Printf("         %s\n", truncBody(sc.Summary, 70))
		}
	}
	return nil
}

func runScanShow(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	scan, err := s.GetScan(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Scan %s\n", scan.ID)
	fmt.Printf("  Project:  %s\n", scan.Project)
	fmt.Printf("  Source:   %s\n", scan.Source)
	fmt.Printf("  Status:   %s\n", strings.ToUpper(scan.Status))
	fmt.Printf("  Started:  %s\n", scan.StartedAt.Format("2006-01-02 15:04:05"))
	if scan.CompletedAt != nil {
		fmt.Printf("  Finished: %s\n", scan.CompletedAt.Format("2006-01-02 15:04:05"))
		d := scan.CompletedAt.Sub(scan.StartedAt)
		fmt.Printf("  Duration: %s\n", d.Round(1e9))
	}
	fmt.Printf("  Entries:  %d\n", scan.EntriesCreated)

	if scan.Summary != "" {
		if scan.SummaryEncrypted {
			fmt.Printf("\n  Summary: [encrypted — unlock session]\n")
		} else {
			fmt.Printf("\n  Summary:\n    %s\n", scan.Summary)
		}
	}

	if scan.Thoughts != "" {
		if scan.ThoughtsEncrypted {
			fmt.Printf("\n  Thoughts: [encrypted — unlock session]\n")
		} else {
			fmt.Printf("\n  Thought log:\n")
			for _, line := range strings.Split(scan.Thoughts, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
	}

	entries, err := s.ScanEntries(scan.ID)
	if err == nil && len(entries) > 0 {
		fmt.Printf("\n  Memories created:\n")
		for _, e := range entries {
			body := truncBody(e.Body, 50)
			if e.BodyEncrypted {
				body = "[encrypted]"
			}
			fmt.Printf("    [%s] %s — %s\n", shortID(e.ID), e.Kind, body)
		}
	}

	return nil
}
