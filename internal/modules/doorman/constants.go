package doorman

const (
	// Docker resource naming (node-scoped; volumes piggyback a suffix).
	instancePrefix = "noxis-"
	instanceLabel  = "noxis.engine"  // container label: the engine kind
	nodeLabel      = "noxis.node"    // container label: the owning node
	desiredLabel   = "noxis.desired" // container label: kind|version (drift compare)
	dataVolSuffix  = "-data"

	// Wire contract: op names crossing master REST <-> slave (control.Request
	// and the slave's runOp switch both honor these).
	OpQuery     = "query"
	OpAlter     = "alter"
	OpRollback  = "rollback"
	OpAnalytics = "analytics"

	// Client binary names; psql is the discriminator for the two CLI dialects.
	clientPostgres = "psql"

	// Client command fragments (postgres psql, mysql/mariadb -N -e).
	clientFlag    = "-c"
	mysqlEvalFlag = "-e"

	// SQL statement fragments for the dry-run transaction wrapper.
	stmtBegin    = "BEGIN;"
	stmtRollback = "ROLLBACK;"

	// Statement kind guards: each op accepts only appropriate statements
	// (verify before execute — a control plane must not run DROP via query).
	readKinds = "SELECT SHOW EXPLAIN DESCRIBE DESC WITH"
	dmlKinds  = "INSERT UPDATE DELETE MERGE REPLACE"
	ddlKinds  = "ALTER CREATE DROP TRUNCATE RENAME GRANT REVOKE COMMENT"
	kindBoth  = readKinds + " " + dmlKinds

	// Supported engine kinds (doorman is SQL-only by design).
	EnginePostgres = "postgres"
	EngineMariaDB  = "mariadb"
	EngineMySQL    = "mysql"
)
