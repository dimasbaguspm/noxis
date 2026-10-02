package slave

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
)

// NewRegisterCmd is `noxis register slave <name>`: one-shot. It registers the
// node with the master and bakes the identity so `noxis run slave` needs no
// env.
func NewRegisterCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "slave <name>",
		Short: "Register this machine as a slave node with the master",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(entity.EnvMaster, entity.EnvGRPC); err != nil {
				return err
			}
			name := args[0]

			req, err := http.NewRequestWithContext(cmd.Context(),
				http.MethodPost, app.Cfg.Master+"/v1/nodes/"+name+"/register", nil)
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

			if err := Save(Identity{ID: name, GRPC: app.Cfg.GRPC}); err != nil {
				return err
			}
			app.Log.InfoContext(cmd.Context(), "slave registered",
				"id", name, "grpc", app.Cfg.GRPC, "identity", Path())
			return nil
		},
	}
}
