package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"noxis/internal/common/config"
	"noxis/internal/core"
	"noxis/internal/entity"
	"noxis/internal/features/doorman"
)

func newAssignCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "assign NODE ENGINE@VERSION [...]",
		Short: "Set a node's desired engines (e.g. noxis assign zeus pg@16 redis@7)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(config.EnvMaster); err != nil {
				return err
			}
			node := args[0]
			spec := &entity.Spec{Node: node}
			for _, raw := range args[1:] {
				name, version, ok := strings.Cut(raw, "@")
				if !ok || version == "" {
					return fmt.Errorf("engine must be NAME@VERSION, got %q", raw)
				}
				if _, ok := doorman.Lookup(name); !ok {
					return fmt.Errorf("unknown engine %q", name)
				}
				spec.Engines = append(spec.Engines, entity.Engine{Name: name, Type: name, Version: version})
			}
			conn, err := app.dial(cmd.Context())
			if err != nil {
				return err
			}
			defer conn.Close()
			_, err = core.NewControlClient(conn).SetDesiredState(cmd.Context(),
				&core.SetDesiredStateRequest{Node: node, Spec: core.SpecToProto(spec)})
			if err != nil {
				return err
			}
			app.Log.Info("assigned", "node", node, "engines", strings.Join(args[1:], " "))
			return nil
		},
	}
}