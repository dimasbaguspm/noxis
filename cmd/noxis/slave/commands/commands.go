// Package commands is the slave's CLI surface: one cobra verb per file here
// (`run slave`, `register slave`), thin over the daemon (cmd/noxis/slave)
// and identity (internal/repositories/slave).
package commands

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"noxis/cmd/noxis/slave"
	"noxis/internal/entity"
	"noxis/internal/repositories/master"
	reposlave "noxis/internal/repositories/slave"
)

// NewRunCmd is `noxis run slave <id>`: starts the daemon from the identity
// file written by register. The id must match the identity.
func NewRunCmd(log *slog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "slave [id]",
		Short: "Run the slave worker daemon for this machine (needs registered identity)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := reposlave.Load()
			if err != nil {
				return err
			}
			if len(args) == 1 && args[0] != id.ID {
				return fmt.Errorf("identity is %q, asked for %q", id.ID, args[0])
			}
			return slave.Run(cmd.Context(), log, id)
		},
	}
}

// NewRegisterCmd is `noxis register slave <name>`: one-shot. It registers the
// node with the master and bakes the identity so `noxis run slave` needs no
// env.
func NewRegisterCmd(log *slog.Logger, cfg entity.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "slave <name>",
		Short: "Register this machine as a slave node with the master",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cfg.Require(entity.EnvMaster, entity.EnvBroker, entity.EnvTopicPrefix); err != nil {
				return err
			}
			name := args[0]
			if err := master.NewClient(cfg.Master).RegisterNode(cmd.Context(), name); err != nil {
				return fmt.Errorf("register with master: %w", err)
			}
			if err := reposlave.Save(reposlave.Identity{
				ID: name, Broker: cfg.Broker, TopicPrefix: cfg.TopicPrefix, Master: cfg.Master,
			}); err != nil {
				return err
			}
			log.InfoContext(cmd.Context(), "slave registered",
				"id", name, "broker", cfg.Broker, "topic_prefix", cfg.TopicPrefix, "master", cfg.Master, "identity", reposlave.Path())
			return nil
		},
	}
}
