package model

import (
	"testing"
)

func TestValidKinds(t *testing.T) {
	expected := []EntryKind{
		KindNote, KindDecision, KindBug, KindSolution,
		KindContext, KindSnippet, KindLearning,
	}
	for _, k := range expected {
		if !ValidKinds[k] {
			t.Errorf("ValidKinds[%q] = false, want true", k)
		}
	}

	if ValidKinds["invalid"] {
		t.Error("ValidKinds[\"invalid\"] = true, want false")
	}
	if ValidKinds[""] {
		t.Error("ValidKinds[\"\"] = true, want false")
	}
}

func TestValidKindsCount(t *testing.T) {
	if len(ValidKinds) != 7 {
		t.Errorf("len(ValidKinds) = %d, want 7", len(ValidKinds))
	}
}

func TestValidRelations(t *testing.T) {
	expected := []LinkRelation{
		RelRelatesTo, RelCausedBy, RelSupersedes, RelSolvedBy,
		RelDependsOn, RelPartOf, RelDerivedFrom,
	}
	for _, r := range expected {
		if !ValidRelations[r] {
			t.Errorf("ValidRelations[%q] = false, want true", r)
		}
	}

	if ValidRelations["invalid"] {
		t.Error("ValidRelations[\"invalid\"] = true, want false")
	}
}

func TestValidRelationsCount(t *testing.T) {
	if len(ValidRelations) != 7 {
		t.Errorf("len(ValidRelations) = %d, want 7", len(ValidRelations))
	}
}
