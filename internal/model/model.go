package model

import "time"

type EntryKind string

const (
	KindNote     EntryKind = "note"
	KindDecision EntryKind = "decision"
	KindBug      EntryKind = "bug"
	KindSolution EntryKind = "solution"
	KindContext  EntryKind = "context"
	KindSnippet  EntryKind = "snippet"
	KindLearning EntryKind = "learning"
)

var ValidKinds = map[EntryKind]bool{
	KindNote: true, KindDecision: true, KindBug: true,
	KindSolution: true, KindContext: true, KindSnippet: true,
	KindLearning: true,
}

type LinkRelation string

const (
	RelRelatesTo   LinkRelation = "relates_to"
	RelCausedBy    LinkRelation = "caused_by"
	RelSupersedes  LinkRelation = "supersedes"
	RelSolvedBy    LinkRelation = "solved_by"
	RelDependsOn   LinkRelation = "depends_on"
	RelPartOf      LinkRelation = "part_of"
	RelDerivedFrom LinkRelation = "derived_from"
)

var ValidRelations = map[LinkRelation]bool{
	RelRelatesTo: true, RelCausedBy: true, RelSupersedes: true,
	RelSolvedBy: true, RelDependsOn: true, RelPartOf: true,
	RelDerivedFrom: true,
}

type Entry struct {
	ID            string     `json:"id"`
	Kind          EntryKind  `json:"kind"`
	Title         string     `json:"title,omitempty"`
	Body          string     `json:"body"`
	BodyEncrypted bool       `json:"body_encrypted"`
	Importance    int        `json:"importance"`
	Source        string     `json:"source"`
	Project       string     `json:"project,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
}

type Tag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Count     int       `json:"count,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Link struct {
	ID          string       `json:"id"`
	FromEntryID string       `json:"from_entry_id"`
	ToEntryID   string       `json:"to_entry_id"`
	Relation    LinkRelation `json:"relation"`
	Weight      float64      `json:"weight"`
	Metadata    string       `json:"metadata,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

type Scan struct {
	ID                string     `json:"id"`
	Project           string     `json:"project"`
	Source            string     `json:"source"`
	Status            string     `json:"status"`
	Summary           string     `json:"summary,omitempty"`
	SummaryEncrypted  bool       `json:"summary_encrypted"`
	Thoughts          string     `json:"thoughts,omitempty"`
	ThoughtsEncrypted bool       `json:"thoughts_encrypted"`
	EntriesCreated    int        `json:"entries_created"`
	StartedAt         time.Time  `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

type Export struct {
	Version    int       `json:"version"`
	Entries    []Entry   `json:"entries"`
	Tags       []Tag     `json:"tags"`
	Links      []Link    `json:"links"`
	ExportedAt time.Time `json:"exported_at"`
}
