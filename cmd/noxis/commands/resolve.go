package commands

import (
	"context"
	"fmt"
	"noxis/internal/client"
	"noxis/internal/common/ref"
	"noxis/internal/domain"
	"strings"
)

// A reference is thirty-six characters and a person is standing at a terminal looking at a NAME. Every
// command that takes one accepts, in this order: the reference itself, the name a list prints, or a unique
// prefix of either. Anything ambiguous is refused and says what it matched — guessing would act on the
// wrong instance, which is the one mistake this product cannot make for you.

// resolveLimit is how many rows a resolution reads. One page of 200 is more instances than one deployment
// governs; past that, a person needs the reference anyway.
const resolveLimit = 200

// candidate is one row a typed value can match: the reference it resolves to and the name to match.
type candidate struct {
	ref  string
	name string
}

// TargetRef resolves what was typed into a target reference.
func TargetRef(ctx context.Context, api *client.Client, typed string) (string, error) {
	if ref.Valid(typed) {
		return typed, nil
	}
	page, err := api.ListTargets(ctx, client.ListTargetsInput{Limit: resolveLimit})
	if err != nil {
		return "", err
	}
	all := make([]candidate, 0, len(page.Items))
	for _, target := range page.Items {
		all = append(all, candidate{ref: target.Ref, name: target.Name})
	}
	return match("target", typed, all)
}

// ProposalRef resolves what was typed into a proposal reference. A proposal has no name, so a prefix is
// what a person has: the seven characters a list prints are enough to be unambiguous in practice.
func ProposalRef(ctx context.Context, api *client.Client, typed string) (string, error) {
	if ref.Valid(typed) {
		return typed, nil
	}
	page, err := api.ListProposals(ctx, resolveLimit, "", "", "", "", false)
	if err != nil {
		return "", err
	}
	all := make([]candidate, 0, len(page.Items))
	for _, proposal := range page.Items {
		all = append(all, candidate{ref: proposal.Ref, name: proposal.Title})
	}
	return match("proposal", typed, all)
}

// match picks the one row typed names. A name wins over a prefix; a prefix that matches more than one row
// is an error that lists them, because a command that guesses is a command that changed the wrong schema.
func match(kind, typed string, all []candidate) (string, error) {
	if typed == "" {
		return "", fmt.Errorf("%w: name a %s, or leave the argument out", domain.ErrInvalid, kind)
	}
	byName := make([]candidate, 0, 1)
	byPrefix := make([]candidate, 0, 1)
	for _, c := range all {
		switch {
		case strings.EqualFold(c.name, typed):
			byName = append(byName, c)
		case strings.HasPrefix(c.ref, typed):
			byPrefix = append(byPrefix, c)
		}
	}
	found := byName
	if len(found) == 0 {
		found = byPrefix
	}

	switch len(found) {
	case 1:
		return found[0].ref, nil
	case 0:
		return "", fmt.Errorf("%w: no %s answers to %q; `noxis %s%s --json` lists what does",
			domain.ErrNotFound, kind, typed, kind, plural(kind))
	default:
		names := make([]string, 0, len(found))
		for _, c := range found {
			names = append(names, c.name+" ("+c.ref[:8]+")")
		}
		return "", fmt.Errorf("%w: %q matches %d %ss — %s; type more of the reference",
			domain.ErrInvalid, typed, len(found), kind, strings.Join(names, ", "))
	}
}

// plural is the command group a list lives under, so the hint it prints is a command that exists.
func plural(kind string) string {
	if kind == "proposal" {
		return "s"
	}
	return " ls"
}
