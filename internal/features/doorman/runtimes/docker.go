package runtimes

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"noxis/internal/entity"
)

// docker contract constants: labels, name prefix, exec client recipes.
const (
	LabelManaged = "noxis=true"     // marks every container doorman owns
	LabelNode    = "noxis.node="    // node scoping (shared docker hosts)
	LabelDesired = "noxis.desired=" // stamped shape: version/type/ports/env
	InspectFmt   = "{{.Name}}\t{{.State.Status}}\t{{index .Config.Labels \"noxis.desired\"}}"
)

// sqlClient describes how to run a read query inside an engine container.
type sqlClient struct {
	bin   string   // client binary inside the image
	pwEnv string   // env var the client reads its password from
	pwVar string   // env var the engine was created with
	pre   []string // fixed args (user/db)
	flags []string // output shape: tab-separated raw tuples
}

const (
	psqlUser = "-U"
	psqlDB   = "-d"
)

var sqlClients = map[string]sqlClient{
	"postgres": {bin: "psql", pwEnv: "PGPASSWORD", pwVar: "POSTGRES_PASSWORD",
		pre: []string{psqlUser, "postgres", psqlDB, "postgres"}, flags: []string{"-A", "-t"}},
	"mysql": {bin: "mysql", pwEnv: "MYSQL_PWD", pwVar: "MYSQL_ROOT_PASSWORD",
		flags: []string{"-N"}},
	"mariadb": {bin: "mariadb", pwEnv: "MYSQL_PWD", pwVar: "MARIADB_ROOT_PASSWORD",
		flags: []string{"-N"}},
}

// Exec shells out to the docker CLI: zero SDK dependency, every action
// readable as a docker command, and swappable in tests.
var Exec = func(_ context.Context, args ...string) (string, error) {
	out, err := exec.Command("docker", args...).CombinedOutput()
	out = []byte(strings.TrimSpace(string(out)))
	if err != nil && len(out) > 0 {
		err = fmt.Errorf("%v: %s", err, string(out))
	}
	return string(out), err
}

type Observed struct {
	State   string
	Desired string // the noxis.desired label: what this container was created to be
}

// List returns this node's containers keyed by name. Node-scoped by label;
// inspect (not ps) because ps --format exposes labels as a raw slice.
func List(ctx context.Context, node string) (map[string]Observed, error) {
	ids, err := Exec(ctx, "ps", "-aq",
		"--filter", "label="+LabelManaged,
		"--filter", "label="+LabelNode+node)
	if err != nil {
		return nil, err
	}
	found := map[string]Observed{}
	if strings.TrimSpace(ids) == "" {
		return found, nil
	}
	args := []string{"inspect", "--format", InspectFmt}
	args = append(args, strings.Fields(ids)...)
	out, err := Exec(ctx, args...)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) == 3 {
			found[strings.TrimPrefix(parts[0], "/")] = Observed{State: parts[1], Desired: parts[2]}
		}
	}
	return found, nil
}

// Run creates the container, stamping its exact shape into noxis.desired.
func Run(ctx context.Context, node string, e entity.Engine) error {
	d, ok := Lookup(e.Type)
	if !ok {
		return fmt.Errorf("unknown engine type %q", e.Type)
	}
	args := []string{"run", "-d",
		"--name", ContainerName(node, e.Name),
		"--label", LabelManaged,
		"--label", LabelNode + node,
		"--restart", "unless-stopped",
	}
	for host, container := range mergedPorts(e, d) {
		args = append(args, "-p", host+":"+container)
	}
	for _, kv := range sortedEnv(mergedEnv(e, d)) {
		args = append(args, "-e", kv[0]+"="+kv[1])
	}
	desired, err := DesiredLabel(e)
	if err != nil {
		return err
	}
	args = append(args, "--label", LabelDesired+desired)
	args = append(args, d.image(e.Version))
	_, err = Exec(ctx, args...)
	return err
}

func Remove(ctx context.Context, name string) error {
	_, err := Exec(ctx, "rm", "-f", name)
	return err
}

func Start(ctx context.Context, name string) error {
	_, err := Exec(ctx, "start", name)
	return err
}

// ExecSQL runs a query in the container via its SQL client; credentials come
// from the same merged env the container was created with.
func ExecSQL(ctx context.Context, container string, e entity.Engine, sql string) (string, error) {
	d, ok := Lookup(e.Type)
	if !ok {
		return "", fmt.Errorf("unknown engine type %q", e.Type)
	}
	client, ok := sqlClients[e.Type]
	if !ok {
		return "", fmt.Errorf("query not supported for %s yet (SQL engines only)", e.Type)
	}
	// IP family note: default hostname resolution targets localhost in the
	// container; DNS names from the engine's own network are inherited.
	args := []string{"exec", "-e", client.pwEnv + "=" + mergedEnv(e, d)[client.pwVar], container}
	args = append(args, client.bin)
	args = append(args, client.pre...)
	args = append(args, client.flags...)
	args = append(args, "-c", sql)
	out, err := Exec(ctx, args...)
	if err != nil {
		return "", err
	}
	return out, nil
}

func sortedEnv(env map[string]string) [][2]string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([][2]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, [2]string{k, env[k]})
	}
	return out
}
