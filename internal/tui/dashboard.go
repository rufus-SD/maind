package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/rufus-SD/maind/internal/model"
	"github.com/rufus-SD/maind/internal/store"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	actActionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Width(8)

	actTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	actIDStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	outputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type tickMsg time.Time
type activityMsg []store.Activity
type statsMsg store.Stats

type Model struct {
	store      *store.Store
	name       string
	encrypted  bool
	activities []store.Activity
	stats      store.Stats
	input      textinput.Model
	output     []string
	width      int
	height     int
	ready      bool
	quitting   bool
	lastActID  int
}

func NewModel(s *store.Store, name string, encrypted bool) Model {
	ti := textinput.New()
	ti.Prompt = "  maind> "
	ti.Focus()
	ti.CharLimit = 256
	ti.PromptStyle = promptStyle

	return Model{
		store:     s,
		name:      name,
		encrypted: encrypted,
		input:     ti,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			cmd := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if cmd != "" {
				m.handleCommand(cmd)
				if m.quitting {
					return m, tea.Quit
				}
			}
			return m, nil
		}

	case tickMsg:
		if acts, err := m.store.RecentActivity(30); err == nil {
			m.activities = acts
		}
		m.stats = m.store.Stats()
		return m, tickCmd()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) handleCommand(line string) {
	parts := splitCmd(line)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "exit", "quit", "q", "/bye", "/exit", "/quit":
		m.quitting = true

	case "help", "?":
		m.output = []string{
			"  remember <text>       Store a memory",
			"  recall <query>        Search memories",
			"  list [kind]           List memories",
			"  show <id>             Show detail",
			"  tags                  List tags",
			"  link <from> <to>      Link memories",
			"  forget <id>           Archive",
			"  scan [list|show <id>] Review AI scans",
			"  exit                  Lock and exit",
		}

	case "recall", "search", "s":
		if len(parts) < 2 {
			m.output = []string{"  Usage: recall <query>"}
			return
		}
		query := strings.Join(parts[1:], " ")
		entries, err := m.store.SearchEntries(query, store.SearchOptions{Limit: 10})
		if err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		m.output = nil
		if len(entries) == 0 {
			m.output = []string{"  No memories found."}
			return
		}
		for _, e := range entries {
			body := strings.ReplaceAll(e.Body, "\n", " ")
			if len(body) > 80 {
				body = body[:80] + "..."
			}
			m.output = append(m.output, fmt.Sprintf("  %s %s (%d) %s",
				actIDStyle.Render("["+shortID(e.ID)+"]"),
				string(e.Kind), e.Importance, body))
		}
		m.store.LogActivity("RECALL", fmt.Sprintf("%q → %d found", query, len(entries)), "")

	case "remember", "r":
		if len(parts) < 2 {
			m.output = []string{"  Usage: remember <text>"}
			return
		}
		body := strings.Join(parts[1:], " ")
		entry := &model.Entry{
			Kind:       model.KindNote,
			Body:       body,
			Importance: 5,
			Source:     "cli",
		}
		if err := m.store.CreateEntry(entry); err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		m.store.LogActivity("STORE", fmt.Sprintf("note: %s", trunc(body, 50)), entry.ID)
		m.output = []string{fmt.Sprintf("  Saved %s", actIDStyle.Render("["+shortID(entry.ID)+"]"))}

	case "list", "ls", "l":
		opts := store.ListOptions{Limit: 15, SortBy: "created_at", SortOrder: "DESC"}
		if len(parts) > 1 {
			opts.Kind = parts[1]
		}
		entries, err := m.store.ListEntries(opts)
		if err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		m.output = nil
		for _, e := range entries {
			body := strings.ReplaceAll(e.Body, "\n", " ")
			if len(body) > 70 {
				body = body[:70] + "..."
			}
			m.output = append(m.output, fmt.Sprintf("  %s %s (%d) %s",
				actIDStyle.Render("["+shortID(e.ID)+"]"), string(e.Kind), e.Importance, body))
		}

	case "show":
		if len(parts) < 2 {
			m.output = []string{"  Usage: show <id>"}
			return
		}
		entry, err := m.store.GetEntry(parts[1])
		if err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		m.output = []string{
			fmt.Sprintf("  ID:         %s", entry.ID),
			fmt.Sprintf("  Kind:       %s", entry.Kind),
			fmt.Sprintf("  Importance: %d/10", entry.Importance),
		}
		if len(entry.Tags) > 0 {
			m.output = append(m.output, fmt.Sprintf("  Tags:       %s", strings.Join(entry.Tags, ", ")))
		}
		m.output = append(m.output, fmt.Sprintf("  Created:    %s", entry.CreatedAt.Format("2006-01-02 15:04")))
		m.output = append(m.output, "")
		m.output = append(m.output, "  "+entry.Body)

	case "tags":
		tags, err := m.store.ListTags()
		if err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		m.output = nil
		for _, t := range tags {
			m.output = append(m.output, fmt.Sprintf("  %-20s (%d)", t.Name, t.Count))
		}

	case "link":
		if len(parts) < 3 {
			m.output = []string{"  Usage: link <from> <to> [relation]"}
			return
		}
		rel := model.RelRelatesTo
		if len(parts) > 3 {
			r := model.LinkRelation(parts[3])
			if model.ValidRelations[r] {
				rel = r
			}
		}
		link := &model.Link{FromEntryID: parts[1], ToEntryID: parts[2], Relation: rel, Weight: 1.0}
		if err := m.store.CreateLink(link); err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		m.store.LogActivity("LINK", fmt.Sprintf("%s --%s--> %s", shortID(link.FromEntryID), link.Relation, shortID(link.ToEntryID)), "")
		m.output = []string{fmt.Sprintf("  Linked %s --%s--> %s", shortID(parts[1]), rel, shortID(parts[2]))}

	case "forget":
		if len(parts) < 2 {
			m.output = []string{"  Usage: forget <id>"}
			return
		}
		if err := m.store.SoftDeleteEntry(parts[1]); err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		m.store.LogActivity("ARCHIVE", parts[1], parts[1])
		m.output = []string{fmt.Sprintf("  Archived [%s]", parts[1])}

	case "scan", "scans":
		m.handleScan(parts[1:])

	default:
		m.output = []string{fmt.Sprintf("  Unknown: %s (type 'help')", parts[0])}
	}
}

func (m *Model) handleScan(args []string) {
	sub := "list"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "list", "ls":
		scans, err := m.store.ListScans(10)
		if err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		if len(scans) == 0 {
			m.output = []string{
				"  No scans yet.",
				"",
				"  Tell your AI: \"scan this project\" — it knows the protocol.",
			}
			return
		}
		m.output = nil
		for _, sc := range scans {
			status := sc.Status
			switch status {
			case "running":
				status = "RUNNING"
			case "completed":
				status = "DONE   "
			case "failed":
				status = "FAILED "
			}
			date := sc.StartedAt.Local().Format("01-02 15:04")
			dur := ""
			if sc.CompletedAt != nil {
				d := sc.CompletedAt.Sub(sc.StartedAt)
				if d.Seconds() < 60 {
					dur = fmt.Sprintf(" %ds", int(d.Seconds()))
				} else {
					dur = fmt.Sprintf(" %dm", int(d.Minutes()))
				}
			}
			line := fmt.Sprintf("  %s %s  %s  %-18s  %d entries%s",
				actIDStyle.Render("["+shortID(sc.ID)+"]"),
				status, date, sc.Project, sc.EntriesCreated, dur)
			m.output = append(m.output, line)
			if sc.Summary != "" {
				m.output = append(m.output, fmt.Sprintf("         %s", trunc(sc.Summary, 60)))
			}
		}

	case "show":
		if len(args) < 2 {
			m.output = []string{"  Usage: scan show <id>"}
			return
		}
		scan, err := m.store.GetScan(args[1])
		if err != nil {
			m.output = []string{fmt.Sprintf("  Error: %v", err)}
			return
		}
		m.output = []string{
			fmt.Sprintf("  Scan %s", actIDStyle.Render("["+shortID(scan.ID)+"]")),
			fmt.Sprintf("  Project:  %s", scan.Project),
			fmt.Sprintf("  Source:   %s", scan.Source),
			fmt.Sprintf("  Status:   %s", strings.ToUpper(scan.Status)),
			fmt.Sprintf("  Started:  %s", scan.StartedAt.Local().Format("2006-01-02 15:04:05")),
		}
		if scan.CompletedAt != nil {
			m.output = append(m.output, fmt.Sprintf("  Finished: %s", scan.CompletedAt.Local().Format("2006-01-02 15:04:05")))
			d := scan.CompletedAt.Sub(scan.StartedAt)
			m.output = append(m.output, fmt.Sprintf("  Duration: %s", d.Round(1e9)))
		}
		m.output = append(m.output, fmt.Sprintf("  Entries:  %d", scan.EntriesCreated))

		if scan.Summary != "" {
			if scan.SummaryEncrypted {
				m.output = append(m.output, "", "  Summary: [encrypted]")
			} else {
				m.output = append(m.output, "", "  Summary: "+scan.Summary)
			}
		}
		if scan.Thoughts != "" {
			if scan.ThoughtsEncrypted {
				m.output = append(m.output, "", "  Thoughts: [encrypted]")
			} else {
				m.output = append(m.output, "", "  Thought log:")
				for _, line := range strings.Split(scan.Thoughts, "\n") {
					m.output = append(m.output, "    "+line)
				}
			}
		}

		entries, err := m.store.ScanEntries(scan.ID)
		if err == nil && len(entries) > 0 {
			m.output = append(m.output, "", "  Memories created:")
			for _, e := range entries {
				body := trunc(e.Body, 50)
				if e.BodyEncrypted {
					body = "[encrypted]"
				}
				m.output = append(m.output, fmt.Sprintf("    %s %s — %s",
					actIDStyle.Render("["+shortID(e.ID)+"]"), string(e.Kind), body))
			}
		}

	default:
		m.output = []string{
			"  scan             List recent scans",
			"  scan list        List recent scans",
			"  scan show <id>   Show scan details + thought log",
		}
	}
}

func (m Model) View() string {
	if !m.ready {
		return "  Loading..."
	}
	if m.quitting {
		return ""
	}

	w := m.width - 4
	if w < 40 {
		w = 40
	}

	// Header
	status := "unlocked"
	if m.encrypted {
		status = "encrypted · unlocked"
	}
	header := fmt.Sprintf("  %s  %s\n  %s · %d memories · %d tags · %d links",
		titleStyle.Render("MAIND"),
		dimStyle.Render("— "+m.name),
		dimStyle.Render(status),
		m.stats.Entries, m.stats.Tags, m.stats.Links,
	)

	usable := m.height - 10
	if usable < 6 {
		usable = 6
	}

	outLines := len(m.output)
	maxOut := 0
	maxAct := usable
	if outLines > 0 {
		maxOut = usable * 2 / 3
		if maxOut < 5 {
			maxOut = 5
		}
		if maxOut > outLines {
			maxOut = outLines
		}
		maxAct = usable - maxOut
		if maxAct < 3 {
			maxAct = 3
		}
	}

	// Activity
	actLines := []string{}
	shown := maxAct
	if shown > len(m.activities) {
		shown = len(m.activities)
	}
	for i := shown - 1; i >= 0; i-- {
		a := m.activities[i]
		ts := actTimeStyle.Render(a.CreatedAt.Local().Format("15:04:05"))
		action := actActionStyle.Render(a.Action)
		summary := a.Summary
		if len(summary) > w-25 {
			summary = summary[:w-25] + "..."
		}
		id := ""
		if a.EntryID != "" {
			id = " " + actIDStyle.Render("["+shortID(a.EntryID)+"]")
		}
		actLines = append(actLines, fmt.Sprintf("  %s  %s  %s%s", ts, action, summary, id))
	}
	if len(actLines) == 0 {
		actLines = []string{dimStyle.Render("  No activity yet. Waiting for memories...")}
	}

	actSection := dimStyle.Render("  ── Activity ──") + "\n" + strings.Join(actLines, "\n")

	// Output
	outSection := ""
	if outLines > 0 {
		lines := m.output
		truncated := false
		if len(lines) > maxOut {
			lines = lines[:maxOut]
			truncated = true
		}
		outSection = "\n" + dimStyle.Render("  ── Output ──") + "\n" + outputStyle.Render(strings.Join(lines, "\n"))
		if truncated {
			outSection += "\n" + dimStyle.Render(fmt.Sprintf("  ... %d more lines (resize to see)", outLines-maxOut))
		}
	}

	// Separator + input
	sep := dimStyle.Render("  " + strings.Repeat("─", w))
	input := m.input.View()

	return fmt.Sprintf("\n%s\n\n%s\n%s\n%s\n%s\n", header, actSection, outSection, sep, input)
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

func trunc(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func splitCmd(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		c := line[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
		} else if c == '"' || c == '\'' {
			inQuote = true
			quoteChar = c
		} else if c == ' ' || c == '\t' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
