package doorman

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/repositories/docker"
)

// Doorman executes the SQL responsibilities against the node's instances.
type Doorman struct {
	d interfaces.Docker
}

func New(d interfaces.Docker) *Doorman {
	if d == nil {
		d = docker.New()
	}
	return &Doorman{d: d}
}

// InstanceName is the container name for a node's engine (noxis-<node>-<engine>).
func InstanceName(node, engine string) string { return instancePrefix + node + "-" + engine }

// VolumeName is the persistent volume backing an instance, split by major
// version: same-major updates share the volume (data survives), a major bump
// gets a fresh cluster (old volume preserved for manual migration).
//
// ponytail: cross-major data migration is real DB engineering, not a control
// plane's job — fresh cluster on major bump is the honest default.
func VolumeName(node, engine, version string) string {
	return instancePrefix + node + "-" + engine + "-" + majorVersion(version) + dataVolSuffix
}

// majorVersion is the leading numeric segment ("16.6-alpine" -> "16").
func majorVersion(version string) string {
	if i := strings.IndexAny(version, ".-"); i > 0 {
		version = version[:i]
	} else if i == 0 {
		return "0"
	}
	return version
}

// CreateInstance brings up (or confirms) the node's engine container: image
// pulled if missing, data volume ensured, container run on the node's
// network. Returns a detail string. Image drift (version change) recreates
// the container — the data volume survives, so an engine update keeps its
// data. Idempotent otherwise: same image reports "in sync".
func (u *Doorman) CreateInstance(ctx context.Context, node, network string, eng entity.Engine) (string, error) {
	desc, ok := lookupEngine(eng.Type)
	if !ok {
		return "", fmt.Errorf("unsupported engine %q (sql only: %s)", eng.Type, supportedEngines())
	}
	name := InstanceName(node, eng.Name)
	image := desc.imageOf(eng.Version)
	updated := false
	if ct, err := u.d.ContainerInspect(ctx, name); err == nil {
		if ct.Config.Image == image {
			return "in sync", nil
		}
		// engine update: image changed — replace the container, keep the
		// volume. Stop gracefully first: a force-killed DB leaves a stale
		// pidfile that makes the new run exit immediately.
		if err := u.d.ContainerStop(ctx, name); err != nil {
			return "", err
		}
		if err := u.d.ContainerRemove(ctx, name); err != nil {
			return "", err
		}
		updated = true
	} else if !errors.Is(err, docker.ErrNotFound) {
		return "", err
	}
	ok, err := u.d.ImageExists(ctx, image)
	if err != nil {
		return "", err
	}
	if !ok {
		if err := u.d.ImagePull(ctx, image); err != nil {
			return "", fmt.Errorf("pull %s: %w", image, err)
		}
	}

	vol := VolumeName(node, eng.Name, engineVersion(eng, desc))
	ok, err = u.d.VolumeExists(ctx, vol)
	if err != nil {
		return "", err
	}
	if !ok {
		if err := u.d.VolumeCreate(ctx, vol); err != nil {
			return "", err
		}
	}

	env := merged(desc.env, eng.Env)
	ports := map[string]string{desc.port: desc.port}
	for host, cont := range eng.Ports {
		ports[host] = cont
	}
	if err := u.d.ContainerRun(ctx, docker.ContainerOptions{
		Name:  name,
		Image: image,
		Labels: map[string]string{
			instanceLabel: eng.Type,
			nodeLabel:     node,
			desiredLabel:  eng.Type + "|" + engineVersion(eng, desc),
		},
		Env:          env,
		Ports:        ports,
		Volumes:      []string{vol + ":" + desc.dataDir},
		Network:      network,
		NetworkAlias: eng.Name,
	}); err != nil {
		return "", err
	}
	verb := "created"
	if updated {
		verb = "updated"
	}
	return "instance " + name + " " + verb + " (" + image + ")", nil
}

// verifyKind rejects statements whose first keyword is not allowed.
func verifyKind(sql, allowed string) error {
	kind := firstKeyword(sql)
	if !strings.Contains(" "+allowed+" ", " "+kind+" ") {
		return fmt.Errorf("statement %q not allowed here (only: %s)", kind, allowed)
	}
	return nil
}

func firstKeyword(sql string) string {
	trimmed := strings.TrimSpace(sql)
	if i := strings.IndexAny(trimmed, " \t\r\n"); i >= 0 {
		trimmed = trimmed[:i]
	}
	return strings.ToUpper(strings.Trim(trimmed, "(;"))
}

// Query runs statements against the instance — reads (SELECT/SHOW/EXPLAIN/…)
// and DML (INSERT/UPDATE/DELETE/…) but never DDL (verified first).
func (u *Doorman) Query(ctx context.Context, node string, eng entity.Engine, sql string) (string, error) {
	if err := verifyKind(sql, kindBoth); err != nil {
		return "", err
	}
	desc, ok := lookupEngine(eng.Type)
	if !ok {
		return "", fmt.Errorf("unsupported engine %q", eng.Type)
	}
	return u.exec(ctx, node, eng, clientArgs(desc, sql))
}

// AlterTable applies DDL statements (ALTER/CREATE/DROP/…) to the instance
// — reads and DML are rejected (verified first).
func (u *Doorman) AlterTable(ctx context.Context, node string, eng entity.Engine, sql string) (string, error) {
	if err := verifyKind(sql, ddlKinds); err != nil {
		return "", err
	}
	desc, ok := lookupEngine(eng.Type)
	if !ok {
		return "", fmt.Errorf("unsupported engine %q", eng.Type)
	}
	return u.exec(ctx, node, eng, clientArgs(desc, sql))
}

// RollbackTable dry-runs a change inside an explicit transaction that always
// rolls back: validates the statement without persisting it.
//
// ponytail: true schema restore (undo an applied change from a history)
// needs a DDL changelog — wire that to the UCM module when it lands.
func (u *Doorman) RollbackTable(ctx context.Context, node string, eng entity.Engine, sql string) (string, error) {
	desc, ok := lookupEngine(eng.Type)
	if !ok {
		return "", fmt.Errorf("unsupported engine %q", eng.Type)
	}
	args := txRollback(desc, sql)
	return u.exec(ctx, node, eng, args)
}

// AnalyzeTable refreshes/stats the table (ANALYZE on postgres, ANALYZE TABLE
// on mariadb/mysql).
func (u *Doorman) AnalyzeTable(ctx context.Context, node string, eng entity.Engine, table string) (string, error) {
	desc, ok := lookupEngine(eng.Type)
	if !ok {
		return "", fmt.Errorf("unsupported engine %q", eng.Type)
	}
	args := clientArgs(desc, analyzeStmt(eng.Type, table))
	return u.exec(ctx, node, eng, args)
}

func (u *Doorman) exec(ctx context.Context, node string, eng entity.Engine, args []string) (string, error) {
	out, err := u.d.ContainerExec(ctx, InstanceName(node, eng.Name), args...)
	if err != nil {
		return "", fmt.Errorf("%s: %w", InstanceName(node, eng.Name), err)
	}
	// mysql/mariadb warn about -p on the command line; harmless, drop it
	out = strings.ReplaceAll(out, "mysql: [Warning] Using a password on the command line interface can be insecure.\n", "")
	return strings.TrimSpace(out), nil
}

// clientArgs builds the engine CLI invocation for one statement.
func clientArgs(desc sqlEngine, sql string) []string {
	switch desc.client {
	case clientPostgres:
		// local socket auth is trust inside the official image
		return []string{clientPostgres, "-U", "postgres", "-t", "-A", clientFlag, sql}
	default: // mariadb, mysql
		return []string{desc.client, "-u", "root", "-p" + desc.env[desc.passwordEnv],
			"-h", "127.0.0.1", "-N", mysqlEvalFlag, sql}
	}
}

// txRollback wraps the statement in BEGIN/ROLLBACK for a dry run.
func txRollback(desc sqlEngine, sql string) []string {
	if desc.client == clientPostgres {
		return []string{clientPostgres, "-U", "postgres", "-t", "-A",
			clientFlag, stmtBegin, clientFlag, sql, clientFlag, stmtRollback}
	}
	return []string{desc.client, "-u", "root", "-p" + desc.env[desc.passwordEnv],
		"-h", "127.0.0.1", "-N", mysqlEvalFlag, stmtBegin + " " + sql + " " + stmtRollback}
}

func analyzeStmt(typ, table string) string {
	if typ == EnginePostgres {
		return "ANALYZE " + table
	}
	return "ANALYZE TABLE " + table
}

func engineVersion(eng entity.Engine, desc sqlEngine) string {
	if eng.Version != "" {
		return eng.Version
	}
	return desc.version
}

// supportedEngines lists the engine kinds in the error message.
func supportedEngines() string {
	names := make([]string, 0, len(sqlEngines))
	for kind := range sqlEngines {
		names = append(names, kind)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// merged layers overrides onto defaults.
func merged(base, over map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}
