package usecases

// Documentation-only annotation carriers. The change-submit route takes either
// a SQL change or a MongoDB request; swag cannot express that polymorphism on a
// single operation, so the concrete schema is emitted from here (it has no
// @Router, so it adds no path) and the operation references `object`.

// SwaggerMongoChangeBody documents the MongoDB change-submit body.
// @Param body body mongoChangeBody true "mongodb request"
func SwaggerMongoChangeBody() {}

// The four SQL console ops share one Serve; swag emits one operation per
// function, so each route is declared here.

// OpQuery documents the query route.
// @Summary Run a SQL console op (query)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/query [post]
func OpQuery() {}

// OpAlter documents the alter route.
// @Summary Apply a write directly (DML or DDL)
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/alter [post]
func OpAlter() {}

// OpRollback documents the rollback route.
// @Summary Dry-run statements inside a transaction that always rolls back
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "sql"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/rollback [post]
func OpRollback() {}

// OpAnalyze documents the analyze route.
// @Summary Refresh planner statistics for a table
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param body body opBody true "table"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/analyze [post]
func OpAnalyze() {}
