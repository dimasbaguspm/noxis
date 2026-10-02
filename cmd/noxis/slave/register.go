package slave

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
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

			req, err := http.NewRequestWithContext(cmd.Context(),
				http.MethodPost, cfg.Master+"/v1/nodes/"+name+"/register", nil)
			if err != nil {
				return err
			}
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("register with master: %w", err)
			}
			resp.Body.Close()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("register with master: %s", resp.Status)
			}

			if err := Save(Identity{ID: name, Broker: cfg.Broker}); err != nil {
				return err
			}
			log.InfoContext(cmd.Context(), "slave registered",
				"id", name, "broker", cfg.Broker, "identity", Path())
			return nil
		},
	}
}
