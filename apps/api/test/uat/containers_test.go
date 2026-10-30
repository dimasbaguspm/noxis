package uat

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/modules/memcache"
	"noxis/internal/modules/mongodb"
	"noxis/internal/modules/mysql"
	"noxis/internal/modules/postgres"
)

// startEngine runs one image and returns the host and port a module dials.
func startEngine(t *testing.T, image, port string, env map[string]string, ready wait.Strategy) (host string, mappedPort int) {
	t.Helper()
	if os.Getenv("UAT_STUB") == "1" {
		t.Skip("offline replay: no containers")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	if ready == nil {
		ready = wait.ForListeningPort(port + "/tcp").WithStartupTimeout(90 * time.Second)
	}
	ctx := context.Background()
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{port + "/tcp"},
			Env:          env,
			WaitingFor:   ready,
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start %s: %v", image, err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(context.Background()) })
	host, err = ctr.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mapped, err := ctr.MappedPort(ctx, port+"/tcp")
	if err != nil {
		t.Fatal(err)
	}
	return host, int(mapped.Num())
}

// TestUATSQLModulePostgres proves the SQL module against a real Postgres started
// for the acceptance suite.
func TestUATSQLModulePostgres(t *testing.T) {
	host, port := startEngine(t, "postgres:16-alpine", "5432", map[string]string{
		"POSTGRES_PASSWORD": "uat", "POSTGRES_DB": "postgres",
	}, wait.ForSQL("5432/tcp", "pgx", func(host string, port network.Port) string {
		return fmt.Sprintf("postgres://postgres:uat@%s:%d/postgres?sslmode=disable", host, port.Num())
	}).WithStartupTimeout(90*time.Second))
	conn := entity.Connection{
		Name: "pg", Driver: constants.DriverPostgres, Host: host, Port: port,
		User: "postgres", Password: "uat", Database: "postgres",
		Params: map[string]string{"sslmode": "disable"},
	}
	result, err := postgres.New().Query(context.Background(), conn, "SELECT 1 AS n")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(result.Rows) != 1 || len(result.Columns) != 1 || result.Columns[0] != "n" {
		t.Fatalf("result = %+v", result)
	}
}

// TestUATSQLModuleMySQL proves the MySQL driver against a real server.
func TestUATSQLModuleMySQL(t *testing.T) {
	host, port := startEngine(t, "mysql:8", "3306", map[string]string{"MYSQL_ROOT_PASSWORD": "uat", "MYSQL_ROOT_HOST": "%", "MYSQL_DATABASE": "uat"},
		wait.ForSQL("3306/tcp", "mysql", func(host string, port network.Port) string {
			return fmt.Sprintf("root:uat@tcp(%s:%d)/uat", host, port.Num())
		}).WithStartupTimeout(90*time.Second))
	conn := entity.Connection{
		Name: "my", Driver: constants.DriverMySQL, Host: host, Port: port,
		User: "root", Password: "uat", Database: "uat",
	}
	result, err := mysql.New().Query(context.Background(), conn, "SELECT 1 AS n")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(result.Rows) != 1 || result.Columns[0] != "n" {
		t.Fatalf("result = %+v", result)
	}
}

// TestUATMongoModule proves the Mongo module against a real server.
func TestUATMongoModule(t *testing.T) {
	host, port := startEngine(t, "mongo:7", "27017", nil, nil)
	conn := entity.Connection{
		Name: "mongo", Driver: constants.DriverMongo, Host: host, Port: port, Database: "admin",
	}
	if err := mongodb.New().Ping(context.Background(), conn); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

// TestUATCacheModule proves the cache module against a real Redis.
func TestUATCacheModule(t *testing.T) {
	host, port := startEngine(t, "redis:7-alpine", "6379", nil, nil)
	conn := entity.Connection{
		Name: "cache", Driver: constants.DriverRedis, Host: host, Port: port,
	}
	if err := memcache.New(constants.DriverRedis).Ping(context.Background(), conn); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
