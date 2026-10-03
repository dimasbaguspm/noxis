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

	// Statement kind guards: query is read-only; alter takes every write
	// (DML and DDL). Verified per statement before execute — a control plane
	// must neither run DROP via query nor SELECT via alter.
	// WITH is a write opener (CTEs can front INSERT/UPDATE/DELETE), so it
	// lives with the writes; CTE reads stay plain SELECTs on query.
	readKinds  = "SELECT SHOW EXPLAIN DESCRIBE DESC"
	dmlKinds   = "WITH INSERT UPDATE DELETE MERGE REPLACE"
	ddlKinds   = "ALTER CREATE DROP TRUNCATE RENAME GRANT REVOKE COMMENT"
	writeKinds = dmlKinds + " " + ddlKinds

	// Supported engine kinds (doorman is SQL-only by design).
	EnginePostgres = "postgres"
	EngineMariaDB  = "mariadb"
	EngineMySQL    = "mysql"
)
