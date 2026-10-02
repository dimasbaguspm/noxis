package slave

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
	"noxis/internal/repositories/http"
	reposlave "noxis/internal/repositories/slave"
)

// NewRegisterCmd is `noxis register slave <name>`: one-shot. It registers the
// node with the master and bakes the identity so `noxis run slave` needs no
// env.
func NewRegisterCmd(log *slog.Logger, cfg entity.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "slave <name>",
		Short: "Register this machine as a slave node with the master",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cfg.Require(entity.EnvMaster, entity.EnvBroker); err != nil {
				return err
			}
			name := args[0]
			if err := http.NewClient(cfg.Master).Do(cmd.Context(),
				"POST", "/v1/nodes/"+name+"/register", nil, nil); err != nil {
				return fmt.Errorf("register with master: %w", err)
			}
			if err := reposlave.Save(reposlave.Identity{ID: name, Broker: cfg.Broker}); err != nil {
				return err
			}
			log.InfoContext(cmd.Context(), "slave registered",
				"id", name, "broker", cfg.Broker, "identity", reposlave.Path())
			return nil
		},
	}
}
