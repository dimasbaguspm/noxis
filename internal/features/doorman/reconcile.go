// Package doorman converges a node's slice of desired state against docker.
package doorman

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"noxis/internal/entity"
	"noxis/internal/features/doorman/runtimes"
)

// Reconciler diffs desired state against the live docker runtime — no caches,
// so push + manual sync can never double-apply.
type Reconciler struct {
	Node string
}

func NewReconciler(node string) *Reconciler { return &Reconciler{Node: node} }

// Reconcile applies the diff and returns a human-readable summary.
func (r *Reconciler) Reconcile(ctx context.Context, engines []entity.Engine) (string, error) {
	desired := map[string]entity.Engine{}
	for _, e := range engines {
		if _, err := runtimes.DesiredLabel(e); err != nil { // validates engine type early
			return "", err
		}
		desired[runtimes.ContainerName(r.Node, e.Name)] = e
	}

	actual, err := runtimes.List(ctx, r.Node)
	if err != nil {
		return "", err
	}

	var lines []string
	for name, e := range desired {
		want, _ := runtimes.DesiredLabel(e)
		obs, exists := actual[name]
		switch {
		case !exists:
			if err := runtimes.Run(ctx, r.Node, e); err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("started %s (%s)", e.Name, imageOf(e)))
		case obs.Desired != want:
			// version, port, or env drift -> recreate
			if err := runtimes.Remove(ctx, name); err != nil {
				return "", err
			}
			if err := runtimes.Run(ctx, r.Node, e); err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("recreated %s (%s)", e.Name, imageOf(e)))
		case obs.State != "running":
			if err := runtimes.Start(ctx, name); err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("started %s", e.Name))
		}
	}

	for name := range actual {
		if _, want := desired[name]; !want {
			if err := runtimes.Remove(ctx, name); err != nil {
				return "", err
			}
			lines = append(lines, "removed "+name)
		}
	}

	if len(lines) == 0 {
		return "", nil // in sync
	}
	sort.Strings(lines)
	return strings.Join(lines, "; "), nil
}

func imageOf(e entity.Engine) string {
	if img, ok := runtimes.ImageOf(e); ok {
		return img
	}
	return ""
}
