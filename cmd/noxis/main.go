package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"noxis/internal/core"
	"noxis/internal/store"
)

var masterAddr string

func defaultMaster() string {
	if v := os.Getenv("NOXIS_MASTER"); v != "" {
		return v
	}
	return "localhost:9000"
}

var rootCmd = &cobra.Command{
	Use:   "noxis",
	Short: "Personal multi-machine infra control plane",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&masterAddr, "master", defaultMaster(), "master address (or NOXIS_MASTER env)")
	rootCmd.AddCommand(masterCmd, slaveCmd, assignCmd, statusCmd, syncCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

var masterCmd = &cobra.Command{
	Use:   "master",
	Short: "Run as master (durable desired state + control plane)",
	Run:   runMaster,
}

func runMaster(cmd *cobra.Command, _ []string) {
	addr := cmd.Flag("addr").Value.String()
	dbPath := cmd.Flag("db").Value.String()
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	srv := grpc.NewServer()
	core.RegisterControlServer(srv, core.NewServer(st))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Printf("master serving on %s (db %s)", addr, dbPath)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()
	<-ctx.Done()
	srv.Stop() // immediate: long-lived Watch streams would stall a graceful drain
	log.Print("master stopped")
}

func init() {
	masterCmd.Flags().String("addr", ":9000", "listen address")
	masterCmd.Flags().String("db", "./noxis.db", "sqlite path")
}

var slaveCmd = &cobra.Command{
	Use:   "slave",
	Short: "Run as slave (local reconciler for this node)",
	Run:   runSlave,
}

func runSlave(cmd *cobra.Command, _ []string) {
	node := cmd.Flag("node").Value.String()
	if node == "" {
		log.Fatal("--node is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("slave %q watching %s", node, masterAddr)
	if err := slaveLoop(ctx, node, masterAddr); err != nil {
		log.Fatal(err)
	}
}

func init() {
	slaveCmd.Flags().String("node", "", "this node's id")
}

func dial(ctx context.Context) *grpc.ClientConn {
	conn, err := core.Dial(ctx, masterAddr)
	if err != nil {
		log.Fatalf("connect to master %s: %v", masterAddr, err)
	}
	return conn
}