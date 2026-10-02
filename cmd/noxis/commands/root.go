// Package commands is the CLI surface: one file per command, all registered
// in NewRootCmd from a single injected App.
package commands

import (
	"github.com/spf13/cobra"

	"noxis/internal/entity"
)

func NewRootCmd(app *entity.App) *cobra.Command {
	root := &cobra.Command{
		Use:   "noxis",
		Short: "Personal multi-machine infra control plane",
	}
	root.AddCommand(
		newMasterCmd(app),
		newSlaveCmd(app),
		newAssignCmd(app),
		newStatusCmd(app),
		newSyncCmd(app),
		newQueryCmd(app),
	)
	return root
}
