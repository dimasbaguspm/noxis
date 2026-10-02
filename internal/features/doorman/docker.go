package doorman

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// dockerCmd shells out to the docker CLI: zero SDK dependency, and every
// action is readable as a docker command in logs. Swappable in tests.
var dockerCmd = func(_ context.Context, args ...string) (string, error) {
	out, err := exec.Command("docker", args...).CombinedOutput()
	out = []byte(strings.TrimSpace(string(out)))
	if err != nil && len(out) > 0 {
		err = fmt.Errorf("%v: %s", err, string(out))
	}
	return string(out), err
}

type observed struct {
	image string
	state string
}

// listContainers returns this node's noxis-managed containers keyed by name.
// Scoping by node label keeps siblings on the same docker host from touching
// each other's containers — a slave only ever converges its own slice.
func listContainers(ctx context.Context, node string) (map[string]observed, error) {
	out, err := dockerCmd(ctx, "ps", "-a",
		"--filter", "label=noxis=true",
		"--filter", "label=noxis.node="+node,
		"--format", "{{.Names}}\t{{.Image}}\t{{.State}}")
	if err != nil {
		return nil, err
	}
	found := map[string]observed{}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) == 3 {
			found[parts[0]] = observed{image: parts[1], state: parts[2]}
		}
	}
	return found, nil
}

func removeContainer(ctx context.Context, name string) error {
	_, err := dockerCmd(ctx, "rm", "-f", name)
	return err
}