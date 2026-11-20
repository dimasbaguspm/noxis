package uat

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	_ "github.com/go-sql-driver/mysql" // the engines speak the same protocol the product does
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// The account and database the @slow scenarios use. The images create them; the harness seeds them.
const (
	dbUser     = "noxis"
	dbPassword = "noxispw"
	dbName     = "app"
	dbRootPass = "rootpw"
)

// engine is one container the slow scenarios talk to.
type engine struct {
	image string
	env   map[string]string
}

var engines = map[string]engine{
	"mysql": {
		image: "mysql:8.4",
		env: map[string]string{
			"MYSQL_ROOT_PASSWORD": dbRootPass,
			"MYSQL_DATABASE":      dbName,
			"MYSQL_USER":          dbUser,
			"MYSQL_PASSWORD":      dbPassword,
		},
	},
	"mariadb": {
		image: "mariadb:11.4",
		env: map[string]string{
			"MARIADB_ROOT_PASSWORD": dbRootPass,
			"MARIADB_DATABASE":      dbName,
			"MARIADB_USER":          dbUser,
			"MARIADB_PASSWORD":      dbPassword,
		},
	},
}

// enginesUp boots both engines on ports the runtime picks, seeds them, and returns their addresses.
// The returned function terminates them; it is safe to call twice.
func enginesUp(ctx context.Context) (map[string]string, func(), error) {
	started := make(map[string]testcontainers.Container, len(engines))
	var addrs = make(map[string]string, len(engines))
	cleanup := func() {
		for name, container := range started {
			if err := container.Terminate(context.WithoutCancel(ctx)); err != nil {
				fmt.Printf("uat: terminate %s: %v\n", name, err)
			}
		}
	}

	for name, e := range engines {
		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        e.image,
				ExposedPorts: []string{"3306/tcp"},
				Env:          e.env,
				WaitingFor:   wait.ForListeningPort("3306/tcp").WithStartupTimeout(3 * time.Minute),
			},
			Started: true,
		})
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("uat: start %s: %w", name, err)
		}
		started[name] = container
		host, err := container.Host(ctx)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("uat: host of %s: %w", name, err)
		}
		port, err := container.MappedPort(ctx, "3306/tcp")
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("uat: port of %s: %w", name, err)
		}
		addrs[name] = net.JoinHostPort(host, port.Port())
	}

	if err := seed(ctx, addrs); err != nil {
		cleanup()
		return nil, nil, err
	}
	return addrs, cleanup, nil
}

// seed gives the engines the table the scenarios propose against, and the privilege a rehearsal needs.
// It waits for the root account first: a listening port is not a server that answers yet.
func seed(ctx context.Context, addrs map[string]string) error {
	for name, addr := range addrs {
		if err := withRetry(ctx, func() error { return seedOne(ctx, addr) }); err != nil {
			return fmt.Errorf("uat: seed %s: %w", name, err)
		}
		// The account the steps use can authenticate later than root can seed.
		if err := withRetry(ctx, func() error { return loginCheck(ctx, addr) }); err != nil {
			return fmt.Errorf("uat: log in to %s as %s: %w", name, dbUser, err)
		}
	}
	return nil
}

func seedOne(ctx context.Context, addr string) error {
	pool, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		"root", dbRootPass, addr, dbName))
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			email VARCHAR(120) NOT NULL UNIQUE,
			note VARCHAR(50) NULL
		)`,
		"DELETE FROM users",
		`INSERT INTO users (email, note) VALUES ('a@x', 'one'), ('b@x', NULL), ('c@x', 'three')`,
		// bulk is the table big enough to matter: doubling it eighteen times leaves a quarter of a
		// million rows, which is past the size where apply stops waiting for the answer.
		`CREATE TABLE IF NOT EXISTS bulk (id INT PRIMARY KEY AUTO_INCREMENT, note VARCHAR(20) NULL)`,
		"DELETE FROM bulk",
		"INSERT INTO bulk (note) VALUES (NULL)",
		// The rehearsal creates and drops its own throwaway database, so the account it connects as
		// needs the privilege a real operator would give it.
		fmt.Sprintf("GRANT ALL PRIVILEGES ON *.* TO '%s'@'%%'", dbUser),
	} {
		if _, err := pool.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	for range 18 {
		if _, err := pool.ExecContext(ctx, "INSERT INTO bulk (note) SELECT note FROM bulk"); err != nil {
			return err
		}
	}
	// The estimate in information_schema is the number apply reads; ANALYZE is what makes it true.
	if _, err := pool.ExecContext(ctx, "ANALYZE TABLE bulk"); err != nil {
		return err
	}
	return nil
}

// withRetry runs fn until it succeeds or the deadline passes: a container's port opens before its
// server accepts a login, and both engines restart once during their own initialisation.
func withRetry(ctx context.Context, fn func() error) error {
	deadline := time.Now().Add(3 * time.Minute)
	var last error
	for {
		last = fn()
		if last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return last
		}
		select {
		case <-ctx.Done():
			return last
		case <-time.After(time.Second):
		}
	}
}

// loginCheck confirms the account the steps use can authenticate, which is later than the port
// opening and later than root working.
func loginCheck(ctx context.Context, addr string) error {
	pool, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPassword, addr, dbName))
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()
	var one int
	return pool.QueryRowContext(ctx, "SELECT 1").Scan(&one)
}
