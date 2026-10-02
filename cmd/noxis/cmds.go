package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"noxis/internal/core"
	"noxis/internal/features/doorman"
)

// slaveLoop keeps one Watch stream open, reconciling on every push. On
// disconnect it backs off and reconnects; the master replays the full spec on
// (re)open, so no change is ever missed.
func slaveLoop(ctx context.Context, node, addr string) error {
	for attempt := 0; ; attempt++ {
		if err := watchOnce(ctx, node, addr); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			backoff := time.Duration(1<<min(attempt, 5)) * time.Second
			log.Printf("watch failed: %v; retrying in %s", err, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func watchOnce(ctx context.Context, node, addr string) error {
	conn, err := core.Dial(ctx, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := core.NewControlClient(conn)
	stream, err := client.Watch(ctx, &core.WatchRequest{Node: node})
	if err != nil {
		return err
	}
	log.Printf("connected to %s", addr)
	for {
		spec, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			report := true
			if err == io.EOF {
				report = false
			}
			if report {
				_, _ = client.ReportStatus(ctx, &core.Status{Node: node, State: "error", Detail: "disconnected: " + err.Error()})
			}
			return err
		}
		detail, err := doorman.NewReconciler(node).Reconcile(ctx, spec.Engines)
		state := "ok"
		if err != nil {
			state = "error"
		}
		if _, err := client.ReportStatus(ctx, &core.Status{Node: node, State: state, Detail: detail}); err != nil {
			log.Printf("report status: %v", err)
		}
		if err != nil {
			log.Printf("reconcile error: %v", err)
			continue
		}
		if detail == "" {
			detail = "in sync"
		}
		log.Printf("reconciled: %s", detail)
	}
}

var assignCmd = &cobra.Command{
	Use:   "assign NODE ENGINE@VERSION [...]",
	Short: "Set a node's desired engines (e.g. noxis assign zeus pg@16 redis@7)",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		node := args[0]
		engines := make([]*core.EngineSpec, 0, len(args)-1)
		for _, raw := range args[1:] {
			name, version, ok := strings.Cut(raw, "@")
			if !ok || version == "" {
				log.Fatalf("engine must be NAME@VERSION, got %q", raw)
			}
			if _, ok := doorman.Lookup(name); !ok {
				log.Fatalf("unknown engine %q", name)
			}
			engines = append(engines, &core.EngineSpec{Name: name, Type: name, Version: version})
		}
		conn := dial(cmd.Context())
		defer conn.Close()
		client := core.NewControlClient(conn)
		if _, err := client.SetDesiredState(cmd.Context(),
			&core.SetDesiredStateRequest{Node: node, Spec: &core.DesiredSpec{Engines: engines}}); err != nil {
			log.Fatalf("assign: %v", err)
		}
		fmt.Printf("assigned %s: %s\n", node, strings.Join(args[1:], " "))
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [NODE]",
	Short: "Show reported status of all nodes (or one)",
	Run: func(cmd *cobra.Command, args []string) {
		conn := dial(cmd.Context())
		defer conn.Close()
		client := core.NewControlClient(conn)
		list, err := client.ListStatus(cmd.Context(), &core.Empty{})
		if err != nil {
			log.Fatalf("status: %v", err)
		}
		filter := ""
		if len(args) > 0 {
			filter = args[0]
		}
		fmt.Printf("%-10s %-10s %-14s %s\n", "NODE", "STATE", "LAST SEEN", "DETAIL")
		for _, s := range list.Statuses {
			if filter != "" && s.Node != filter {
				continue
			}
			ago := ""
			if t := time.Unix(s.Ts, 0); !t.IsZero() {
				ago = time.Since(t).Truncate(time.Second).String() + " ago"
			}
			fmt.Printf("%-10s %-10s %-14s %s\n", s.Node, s.State, ago, s.Detail)
		}
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync NODE",
	Short: "Manually pull a node's desired state and reconcile it (fallback)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		node := args[0]
		conn := dial(cmd.Context())
		defer conn.Close()
		client := core.NewControlClient(conn)
		spec, err := client.GetDesiredState(cmd.Context(), &core.NodeId{Id: node})
		if err != nil {
			log.Fatalf("get desired state: %v", err)
		}
		detail, err := doorman.NewReconciler(node).Reconcile(cmd.Context(), spec.Engines)
		if err != nil {
			log.Fatalf("reconcile: %v", err)
		}
		if detail == "" {
			detail = "in sync"
		}
		fmt.Println(detail)
	},
}