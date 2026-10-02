package commands

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
	"noxis/internal/features/doorman/runtimes"
	rpc "noxis/internal/repository/grpc"
)

func newAssignCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "assign NODE ENGINE[@VERSION[:HOSTPORT]] [...]",
		Short: "Set a node's desired engines (e.g. noxis assign zeus pg@16 redis@7:6380)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if err := app.Cfg.Require(entity.EnvMaster); err != nil {
				return err
			}
			spec, err := parseSpec(args)
			if err != nil {
				return err
			}
			conn, err := dial(ctx, app)
			if err != nil {
				return err
			}
			defer conn.Close()
			_, err = rpc.NewControlClient(conn).SetDesiredState(ctx,
				&rpc.SetDesiredStateRequest{Node: spec.Node, Spec: rpc.SpecToProto(spec)})
			if err != nil {
				return err
			}
			app.Log.InfoContext(ctx, "assigned", slog.String("node", spec.Node),
				slog.String("engines", strings.Join(args[1:], " ")))
			return nil
		},
	}
}

// parseSpec turns `NODE TYPE@VERSION[:HOSTPORT] ...` into a spec. Multiple
// engines of the same type get -2/-3 name suffixes so their containers don't
// collide; a host port overrides the engine default.
func parseSpec(args []string) (*entity.Spec, error) {
	spec := &entity.Spec{Node: args[0]}
	used := map[string]int{}
	for _, raw := range args[1:] {
		typeName, rest, ok := strings.Cut(raw, "@")
		if !ok || rest == "" {
			return nil, fmt.Errorf("engine must be NAME@VERSION[:HOSTPORT], got %q", raw)
		}
		version, hostPort, _ := strings.Cut(rest, ":")
		if version == "" {
			return nil, fmt.Errorf("engine must be NAME@VERSION[:HOSTPORT], got %q", raw)
		}
		if _, ok := runtimes.Lookup(typeName); !ok {
			return nil, fmt.Errorf("unknown engine %q", typeName)
		}
		name := typeName
		if used[typeName] > 0 {
			name = fmt.Sprintf("%s-%d", typeName, used[typeName]+1) // second redis -> redis-2
		}
		used[typeName]++
		eng := entity.Engine{Name: name, Type: typeName, Version: version}
		if hostPort != "" {
			containerPort, _ := runtimes.DefaultContainerPort(typeName)
			eng.Ports = map[string]string{hostPort: containerPort}
		}
		spec.Engines = append(spec.Engines, eng)
	}
	return spec, nil
}
