// Package doorman is the SQL database module: create instances, run queries,
// alter tables, dry-run rollbacks, and analyze tables. Limited to SQL
// engines (postgres, mariadb, mysql). Container/network mechanics stay in
// internal/repositories/docker — this module decides what the DB shells do.
package doorman

// sqlEngine describes how to run one SQL engine as a container.
type sqlEngine struct {
	image       string            // base image (version appended at runtime)
	version     string            // default image tag when the spec omits one
	port        string            // default host port mapping target
	client      string            // CLI binary inside the image
	passwordEnv string            // env var holding root password (client auth)
	dataDir     string            // container path to mount the data volume on
	env         map[string]string // default env baked into the container
}

var sqlEngines = map[string]sqlEngine{
	"postgres": {
		image: "postgres", version: "16-alpine", port: "5432", client: "psql",
		passwordEnv: "POSTGRES_PASSWORD", dataDir: "/var/lib/postgresql/data",
		env: map[string]string{"POSTGRES_PASSWORD": "noxis"},
	},
	"mariadb": {
		image: "mariadb", version: "11", port: "3306", client: "mariadb",
		passwordEnv: "MARIADB_ROOT_PASSWORD", dataDir: "/var/lib/mysql",
		env: map[string]string{"MARIADB_ROOT_PASSWORD": "noxis"},
	},
	"mysql": {
		image: "mysql", version: "8", port: "3306", client: "mysql",
		passwordEnv: "MYSQL_ROOT_PASSWORD", dataDir: "/var/lib/mysql",
		env: map[string]string{"MYSQL_ROOT_PASSWORD": "noxis"},
	},
}

func lookupEngine(typ string) (sqlEngine, bool) {
	e, ok := sqlEngines[typ]
	return e, ok
}

// engineImage resolves the spec version or falls back to the engine default.
func (e sqlEngine) imageOf(version string) string {
	if version == "" {
		version = e.version
	}
	return e.image + ":" + version
}
