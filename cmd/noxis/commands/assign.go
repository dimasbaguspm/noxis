package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
	"noxis/internal/features/doorman/runtimes"
)

func newAssignCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "assign NODE ENGINE[@VERSION[:HOSTPORT]] [...]",
		Short: "Set a node's desired engines (e.g. noxis assign zeus pg@16 redis@7:6380)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.Cfg.Require(entity.EnvMaster); err != nil {
				return err
			}
			engines, err := parseEngines(args[1:])
			if err != nil {
				return err
			}
			if err := doJSON(cmd.Context(), app, "PUT",
				"/nodes/"+args[0], map[string]any{"engines": engines}, nil); err != nil {
				return err
			}
			app.Log.InfoContext(cmd.Context(), "assigned",
				"node", args[0], "engines", strings.Join(args[1:], " "))
			return nil
		},
	}
}

func parseEngines(raws []string) ([]entity.Engine, error) {
	var out []entity.Engine
	used := map[string]int{}
	for _, raw := range raws {
		typeName, rest, ok := strings.Cut(raw, "@")
		version, hostPort, _ := strings.Cut(rest, ":")
		if !ok || version == "" {
			return nil, fmt.Errorf("engine must be NAME@VERSION[:HOSTPORT], got %q", raw)
		}
		if _, ok := runtimes.Lookup(typeName); !ok {
			return nil, fmt.Errorf("unknown engine %q", typeName)
		}
		name := typeName
		if used[typeName] > 0 {
			name = fmt.Sprintf("%s-%d", typeName, used[typeName]+1)
		}
		used[typeName]++
		e := entity.Engine{Name: name, Type: typeName, Version: version}
		if hostPort != "" {
			containerPort, _ := runtimes.DefaultContainerPort(typeName)
			e.Ports = map[string]string{hostPort: containerPort}
		}
		out = append(out, e)
	}
	return out, nil
}
