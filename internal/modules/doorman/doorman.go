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

// VolumeName is the persistent volume backing an instance.
func VolumeName(node, engine string) string {
	return instancePrefix + node + "-" + engine + dataVolSuffix
}

// CreateInstance brings up (or confirms) the node's engine container: image
// pulled if missing, data volume ensured, container run on the node's
// network. Returns a detail string. Idempotent — an already-running instance
// reports "in sync".
//
// ponytail: no drift comparison yet — version/port/env changes recreate only
// when instance teardown happens; diff via desiredLabel when it matters.
func (u *Doorman) CreateInstance(ctx context.Context, node, network string, eng entity.Engine) (string, error) {
	desc, ok := lookupEngine(eng.Type)
	if !ok {
		return "", fmt.Errorf("unsupported engine %q (sql only: %s)", eng.Type, supportedEngines())
	}
	name := InstanceName(node, eng.Name)
	if _, err := u.d.ContainerInspect(ctx, name); err == nil {
		return "in sync", nil
	} else if !errors.Is(err, docker.ErrNotFound) {
		return "", err
	}

	image := desc.imageOf(eng.Version)
	ok, err := u.d.ImageExists(ctx, image)
	if err != nil {
		return "", err
	}
	if !ok {
		if err := u.d.ImagePull(ctx, image); err != nil {
			return "", fmt.Errorf("pull %s: %w", image, err)
		}
	}

	vol := VolumeName(node, eng.Name)
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
	return "instance " + name + " created (" + image + ")", nil
}

// Query runs a read against the instance.
func (u *Doorman) Query(ctx context.Context, node string, eng entity.Engine, sql string) (string, error) {
	desc, ok := lookupEngine(eng.Type)
	if !ok {
		return "", fmt.Errorf("unsupported engine %q", eng.Type)
	}
	return u.exec(ctx, node, eng, clientArgs(desc, sql))
}

// AlterTable applies a DDL statement (ALTER TABLE ...) to the instance.
func (u *Doorman) AlterTable(ctx context.Context, node string, eng entity.Engine, sql string) (string, error) {
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
