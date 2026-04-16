package cli

import (
	"fmt"
	"os"

	"github.com/rufus-SD/maind/internal/model"

	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link [from-id] [to-id]",
	Short: "Create a link between two memories",
	Long: `Create a typed, weighted relationship between two memory entries.

Relations: relates_to, caused_by, supersedes, solved_by, depends_on, part_of, derived_from

Examples:
  maind link a1b2c3d4 e5f6g7h8
  maind link a1b2c3d4 e5f6g7h8 --relation solved_by
  maind link a1b2c3d4 e5f6g7h8 --relation caused_by --weight 8`,
	Args: cobra.ExactArgs(2),
	RunE: runLink,
}

var (
	linkRelation string
	linkWeight   float64
)

func init() {
	linkCmd.Flags().StringVarP(&linkRelation, "relation", "r", "relates_to", "relation type")
	linkCmd.Flags().Float64VarP(&linkWeight, "weight", "w", 1.0, "link weight 0.0-10.0")
}

func runLink(cmd *cobra.Command, args []string) error {
	rel := model.LinkRelation(linkRelation)
	if !model.ValidRelations[rel] {
		return fmt.Errorf("invalid relation %q", linkRelation)
	}
	if linkWeight < 0 || linkWeight > 10 {
		return fmt.Errorf("weight must be between 0.0 and 10.0")
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	link := &model.Link{
		FromEntryID: args[0],
		ToEntryID:   args[1],
		Relation:    rel,
		Weight:      linkWeight,
	}

	if err := s.CreateLink(link); err != nil {
		return fmt.Errorf("create link: %w", err)
	}
	s.LogActivity("LINK", fmt.Sprintf("%s --%s--> %s", shortID(link.FromEntryID), link.Relation, shortID(link.ToEntryID)), "")

	fmt.Fprintf(os.Stderr, "Linked [%s] --%s--> [%s]\n", shortID(link.FromEntryID), link.Relation, shortID(link.ToEntryID))
	return nil
}
