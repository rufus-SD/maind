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

	default:
		m.output = []string{fmt.Sprintf("  Unknown: %s (type 'help')", parts[0])}
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

	// Activity
	actLines := []string{}
	maxAct := (m.height - 14) / 2
	if maxAct < 3 {
		maxAct = 3
	}
	if maxAct > len(m.activities) {
		maxAct = len(m.activities)
	}
	for i := maxAct - 1; i >= 0; i-- {
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
	if len(m.output) > 0 {
		maxOut := (m.height - 14) / 2
		if maxOut < 3 {
			maxOut = 3
		}
		lines := m.output
		if len(lines) > maxOut {
			lines = lines[:maxOut]
		}
		outSection = "\n" + dimStyle.Render("  ── Output ──") + "\n" + outputStyle.Render(strings.Join(lines, "\n"))
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
