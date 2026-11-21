package constants

// Tiers are the rehearsals a proposal may ask for, cheapest first. The domain owns the typed list and
// this is the same list as strings, for a client that has to describe the API without importing the
// domain; a test in `domain` fails when the two disagree.
var Tiers = []string{"static", "shadow", "rehearsal"}
