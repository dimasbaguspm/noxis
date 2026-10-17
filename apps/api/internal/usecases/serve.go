package usecases

import (
	"net/http"
	"strconv"

	"noxis/internal/common/config"
	"noxis/internal/common/pagination"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules/mongodb"
)

// --- connections ---

// @Summary List connections (passwords redacted)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /external/v1/engine/{engine}/connections [get]
func (uc ConnectionList) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	out, err := uc.Run(r.Context(), mod)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, map[string]any{"connections": out})
}

// @Summary Create or update a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections [post]
func (uc SaveConnection) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	var c entity.Connection
	if err := decode(r, &c); err != nil {
		fail(w, err)
		return
	}
	out, err := uc.Run(r.Context(), mod, c)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}

// @Summary Get one connection (password redacted)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name} [get]
func (uc Connection) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	out, err := uc.Run(r.Context(), mod, r.PathValue("name"))
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}

// @Summary Delete a connection (and its changes)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name} [delete]
func (uc DeleteConnection) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	if err := uc.Run(r.Context(), mod, r.PathValue("name")); err != nil {
		fail(w, err)
		return
	}
	respondNoContent(w)
}

// @Summary Test that a connection answers
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/test [post]
func (uc TestConnection) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	if err := uc.Run(r.Context(), mod, r.PathValue("name")); err != nil {
		fail(w, err)
		return
	}
	respond(w, entity.StatusResponse{Status: "ok"})
}

// @Summary Introspect a connection's tables and columns
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Success 200 {object} object{tables=[]entity.Table}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/schema [get]
func (uc Schema) Serve(w http.ResponseWriter, r *http.Request) {
	sq, err := uc.Modules.EngineSQL(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	out, err := uc.Run(r.Context(), sq, r.PathValue("name"))
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, map[string]any{"tables": out})
}

// @Summary Preview a table's rows
// @Param engine path string true "engine family (sql)"
// @Param name path string true "connection name"
// @Param table path string true "table name (unqualified)"
// @Param schema query string false "schema (defaults to the driver's)"
// @Param limit query int false "row cap (default 100, max 500)"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/tables/{table}/rows [get]
func (uc Rows) Serve(w http.ResponseWriter, r *http.Request) {
	sq, err := uc.Modules.EngineSQL(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := uc.Run(r.Context(), sq, r.PathValue("name"),
		r.URL.Query().Get("schema"), r.PathValue("table"), limit)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}

// opBody is the console op payload: sql for query/alter/rollback, table for
// analyze.
type opBody struct {
	SQL   string `json:"sql,omitempty"`
	Table string `json:"table,omitempty"`
}

// opRoutes pins each op route to its operation name, read from the matched
// pattern (a request's Pattern is the route it hit).
var opRoutes = map[string]string{
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/query":    constants.OpQuery,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/alter":    constants.OpAlter,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/rollback": constants.OpRollback,
	"POST " + constants.APIPrefix + "/engine/{engine}/connections/{name}/analyze":  constants.OpAnalyze,
}

func (uc Op) Serve(w http.ResponseWriter, r *http.Request) {
	sq, err := uc.Modules.EngineSQL(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	var body opBody
	if err := decode(r, &body); err != nil {
		fail(w, err)
		return
	}
	out, err := uc.Run(r.Context(), sq, r.PathValue("name"), opRoutes[r.Pattern], body.SQL, body.Table)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}

// --- permissions ---

// memberUser is one pickable account: the subject stored on a permission and a
// human label for the console.
type memberUser struct {
	Subject string `json:"subject"`
	Label   string `json:"label"`
}

// @Summary The known groups and user accounts for permission editing
// @Success 200 {object} object{groups=[]string,users=[]memberUser}
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/config/members [get]
func (uc Members) Serve(w http.ResponseWriter, r *http.Request) {
	groups, users, err := uc.Run(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	pickable := make([]memberUser, 0, len(users))
	for _, u := range users {
		label := u.Username
		if label == "" {
			label = u.Subject
		}
		pickable = append(pickable, memberUser{Subject: u.Subject, Label: label})
	}
	respond(w, map[string]any{"groups": groups, "users": pickable})
}

// @Summary List a connection's permissions
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} object{permissions=[]entity.Permission}
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [get]
func (uc ListPermissions) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	out, err := uc.Run(r.Context(), mod, r.PathValue("name"))
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, map[string]any{"permissions": out})
}

// permissionBody is the upsert payload: one member at one level.
type permissionBody struct {
	Member     string `json:"member"`
	MemberType string `json:"memberType"`
	Level      string `json:"level"`
}

// @Summary Grant a user or group a level on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param body body permissionBody true "member, member type and level"
// @Success 200 {object} entity.Permission
// @Failure 400 {object} entity.ErrorResponse
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [put]
func (uc SetPermission) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	var body permissionBody
	if err := decode(r, &body); err != nil {
		fail(w, err)
		return
	}
	out, err := uc.Run(r.Context(), mod, r.PathValue("name"), body.Member, body.MemberType, body.Level)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}

// @Summary Revoke a member's permission on a connection
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param member query string true "user subject or group name"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/permissions [delete]
func (uc DeletePermission) Serve(w http.ResponseWriter, r *http.Request) {
	mod, err := uc.Modules.Engine(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	if err := uc.Run(r.Context(), mod, r.PathValue("name"), r.URL.Query().Get("member")); err != nil {
		fail(w, err)
		return
	}
	respondNoContent(w)
}

// --- instances ---

// @Summary List instances across engine families (passwords redacted)
// @Param name query string false "substring match on name"
// @Param driver query string false "exact driver match"
// @Param host query string false "substring match on host"
// @Param page query int false "page number, starting at 1"
// @Param perPage query int false "items per page (default 20, max 100)"
// @Success 200 {object} object{instances=[]entity.Connection,total=int}
// @Router /external/v1/instances [get]
func (uc ListInstances) Serve(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := uc.Run(r.Context(), q.Get("name"), q.Get("driver"), q.Get("host"))
	if err != nil {
		fail(w, err)
		return
	}
	page, perPage := pagination.Params(q.Get("page"), q.Get("perPage"), constants.DefaultPageSize)
	respond(w, map[string]any{
		"instances": pagination.Slice(out, page, perPage),
		"total":     len(out),
	})
}

// @Summary Get one instance across engine families (password redacted)
// @Param id path string true "instance id"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/instances/{id} [get]
func (uc Instance) Serve(w http.ResponseWriter, r *http.Request) {
	out, err := uc.Run(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}

// passwordBody is the rotate-password payload.
type passwordBody struct {
	Password string `json:"password"`
}

// @Summary Rotate an instance's password (identity fields stay locked)
// @Param id path string true "instance id"
// @Param body body passwordBody true "new password"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 403 {object} entity.ErrorResponse
// @Router /external/v1/instances/{id}/password [post]
func (uc RotatePassword) Serve(w http.ResponseWriter, r *http.Request) {
	var body passwordBody
	if err := decode(r, &body); err != nil {
		fail(w, err)
		return
	}
	if err := uc.Run(r.Context(), r.PathValue("id"), body.Password); err != nil {
		fail(w, err)
		return
	}
	respond(w, entity.StatusResponse{Status: "ok"})
}

// --- identity ---

// MeResponse is the caller plus the effective level it holds on each readable
// connection, so the console can gate actions without guessing.
type MeResponse struct {
	entity.Identity
	Connections map[string]string `json:"connections"`
}

// @Summary The authenticated caller and its per-connection levels
// @Success 200 {object} MeResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/me [get]
func (uc ConnectionLevels) Serve(w http.ResponseWriter, r *http.Request) {
	p, ok := Principal(r.Context())
	if !ok {
		fail(w, constants.ErrUnauthorized)
		return
	}
	levels, err := uc.Run(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, MeResponse{Identity: p, Connections: levels})
}

// --- auth ---

// AuthConfig is the public auth configuration the console bootstraps from.
type AuthConfig struct {
	Auth config.AuthInfo
}

// @Summary Public auth configuration (console bootstrap)
// @Success 200 {object} config.AuthInfo
// @Router /external/v1/auth/config [get]
func (uc AuthConfig) Serve(w http.ResponseWriter, _ *http.Request) { respond(w, uc.Auth) }

// loginRequest is the local credentials payload.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse is the token the console stores and the account it belongs to.
type loginResponse struct {
	Token string      `json:"token"`
	User  entity.User `json:"user"`
}

// Login authenticates a local account.
type Login struct {
	RegistryDeps
}

// @Summary Log in with a local account
// @Accept json
// @Param credentials body loginRequest true "username and password"
// @Success 200 {object} loginResponse
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/login [post]
func (uc Login) Serve(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := decode(r, &body); err != nil {
		fail(w, err)
		return
	}
	token, user, err := uc.Modules.Auth().Login(r.Context(), body.Username, body.Password)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, loginResponse{Token: token, User: user})
}

// Logout ends the caller's session.
type Logout struct {
	RegistryDeps
}

// @Summary Log out the current local account
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/logout [post]
func (uc Logout) Serve(w http.ResponseWriter, r *http.Request) {
	p, ok := Principal(r.Context())
	if !ok {
		fail(w, constants.ErrUnauthorized)
		return
	}
	if err := uc.Modules.Auth().Logout(r.Context(), p.Subject); err != nil {
		fail(w, err)
		return
	}
	respondNoContent(w)
}

// changePasswordRequest is the self-serve password payload.
type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// ChangePassword lets a local account change its own password.
type ChangePassword struct {
	RegistryDeps
}

// @Summary Change the current local account's password
// @Accept json
// @Param passwords body changePasswordRequest true "old and new password"
// @Success 204
// @Failure 401 {object} entity.ErrorResponse
// @Router /external/v1/auth/password [post]
func (uc ChangePassword) Serve(w http.ResponseWriter, r *http.Request) {
	p, ok := Principal(r.Context())
	if !ok {
		fail(w, constants.ErrUnauthorized)
		return
	}
	if entity.LocalUsername(p.Subject) == "" {
		fail(w, errLocalAccountRequired)
		return
	}
	var body changePasswordRequest
	if err := decode(r, &body); err != nil {
		fail(w, err)
		return
	}
	if err := uc.Modules.Auth().ChangePassword(r.Context(), p.Subject, body.OldPassword, body.NewPassword); err != nil {
		fail(w, err)
		return
	}
	respondNoContent(w)
}

// --- configuration ---

// DriverInfo is one entry of the driver catalog the console reads to build its
// pickers. A driver is code, not data, so the catalog is hardcoded Go-side and
// this endpoint is the only source: no UI constant drifts from the server.
type DriverInfo struct {
	Driver string `json:"driver"`
	Family string `json:"family"`
	Label  string `json:"label"`
}

var driverCatalog = []DriverInfo{
	{constants.DriverPostgres, constants.FamilySQL, "PostgreSQL"},
	{constants.DriverMySQL, constants.FamilySQL, "MySQL / MariaDB"},
	{constants.DriverSQLite, constants.FamilySQL, "SQLite"},
	{constants.DriverRedis, constants.FamilyMemcache, "Redis"},
	{constants.DriverValkey, constants.FamilyMemcache, "Valkey"},
	{constants.DriverMongo, constants.FamilyMongo, "MongoDB"},
}

// Drivers serves the driver catalog.
type Drivers struct{}

// @Summary List the drivers the console can register
// @Success 200 {array} DriverInfo
// @Router /external/v1/config/drivers [get]
func (Drivers) Serve(w http.ResponseWriter, _ *http.Request) { respond(w, driverCatalog) }

// --- memory cache ---

// @Summary Browse keys by pattern (bounded scan, with type, TTL and size)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param pattern query string false "glob pattern (default *)"
// @Param limit query int false "scan cap (default 500, max 5000)"
// @Success 200 {object} entity.KeyPage
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/keys [get]
func (uc CacheKeys) Serve(w http.ResponseWriter, r *http.Request) {
	mc, err := uc.Modules.EngineMemcache(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	c, err := rawConn(r.Context(), uc.Connections, uc.Permissions, uc.Modules, mc, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		fail(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := mc.Keys(r.Context(), *c, r.URL.Query().Get("pattern"), limit)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}

// @Summary Inspect one key's value (rendered per its type)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param key query string true "key name"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/value [get]
func (uc CacheValue) Serve(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		fail(w, errKeyRequired)
		return
	}
	mc, err := uc.Modules.EngineMemcache(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	c, err := rawConn(r.Context(), uc.Connections, uc.Permissions, uc.Modules, mc, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		fail(w, err)
		return
	}
	out, err := mc.Value(r.Context(), *c, key)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}

// cacheKeysBody is the remove-keys payload.
type cacheKeysBody struct {
	Keys []string `json:"keys"`
}

// @Summary Remove keys (the selected ones only)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param body body cacheKeysBody true "keys to remove"
// @Success 200 {object} object{deleted=int}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/keys [delete]
func (uc CacheDelete) Serve(w http.ResponseWriter, r *http.Request) {
	var body cacheKeysBody
	if err := decode(r, &body); err != nil {
		fail(w, err)
		return
	}
	mc, err := uc.Modules.EngineMemcache(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	c, err := rawConn(r.Context(), uc.Connections, uc.Permissions, uc.Modules, mc, r.PathValue("name"), entity.LevelAdmin)
	if err != nil {
		fail(w, err)
		return
	}
	deleted, err := mc.Delete(r.Context(), *c, body.Keys)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, map[string]any{"deleted": deleted})
}

// cacheFlushBody is the flush payload: confirm must repeat the connection name.
type cacheFlushBody struct {
	Confirm string `json:"confirm"`
}

// @Summary Empty the connected database (FLUSHDB, never FLUSHALL)
// @Param engine path string true "engine family (memcache)"
// @Param name path string true "connection name"
// @Param body body cacheFlushBody true "confirm must repeat the connection name"
// @Success 200 {object} entity.StatusResponse
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/flush [post]
func (uc CacheFlush) Serve(w http.ResponseWriter, r *http.Request) {
	var body cacheFlushBody
	if err := decode(r, &body); err != nil {
		fail(w, err)
		return
	}
	mc, err := uc.Modules.EngineMemcache(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	c, err := rawConn(r.Context(), uc.Connections, uc.Permissions, uc.Modules, mc, r.PathValue("name"), entity.LevelAdmin)
	if err != nil {
		fail(w, err)
		return
	}
	if body.Confirm != c.Name {
		fail(w, errFlushConfirm(c.Name))
		return
	}
	if err := mc.Flush(r.Context(), *c); err != nil {
		fail(w, err)
		return
	}
	respond(w, entity.StatusResponse{Status: "flushed"})
}

// --- mongodb explorer ---

// @Summary List databases with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} object{databases=[]entity.Database}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/databases [get]
func (uc MongoDatabases) Serve(w http.ResponseWriter, r *http.Request) {
	m, err := uc.Modules.EngineMongo(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	c, err := rawConn(r.Context(), uc.Connections, uc.Permissions, uc.Modules, m, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		fail(w, err)
		return
	}
	out, err := m.Databases(r.Context(), *c)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, map[string]any{"databases": out})
}

// @Summary List a database's collections with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Param database query string true "database name"
// @Success 200 {object} object{collections=[]entity.Collection}
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/collections [get]
func (uc MongoCollections) Serve(w http.ResponseWriter, r *http.Request) {
	m, err := uc.Modules.EngineMongo(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	c, err := rawConn(r.Context(), uc.Connections, uc.Permissions, uc.Modules, m, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		fail(w, err)
		return
	}
	out, err := m.Collections(r.Context(), *c, r.URL.Query().Get("database"))
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, map[string]any{"collections": out})
}

// @Summary Run a read request (find | aggregate | count)
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Param body body entity.MongoRequest true "op, database, collection, filter and value as Extended JSON"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{engine}/connections/{name}/requests [post]
func (uc MongoRun) Serve(w http.ResponseWriter, r *http.Request) {
	var req interfaces.MongoRequest
	if err := decode(r, &req); err != nil {
		fail(w, err)
		return
	}
	m, err := uc.Modules.EngineMongo(r.PathValue("engine"))
	if err != nil {
		fail(w, err)
		return
	}
	// a write op is refused by Run, but it must still not be handed to a
	// read-only caller
	level := entity.LevelRead
	if mongodb.IsWrite(req.Op) {
		level = entity.LevelWrite
	}
	c, err := rawConn(r.Context(), uc.Connections, uc.Permissions, uc.Modules, m, r.PathValue("name"), level)
	if err != nil {
		fail(w, err)
		return
	}
	out, err := m.Run(r.Context(), *c, req)
	if err != nil {
		fail(w, err)
		return
	}
	respond(w, out)
}
