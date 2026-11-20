# [api]: probe a target and declare what its engine can do

Plan issue #751, branch `feat/751-probe`.

## What problem does this solve?

A stored target is a set of strings until something connects to it. Without a probe, "registered" means only that someone typed a hostname, and the server cannot say whether it speaks to MySQL 8 or MariaDB 11 — which is the difference the rehearsal depends on.

## What is the user facing?

`POST /v1/targets/{ref}/probe` connects once, records the server version and flavour on the target, and answers the capabilities it derived. A target it cannot reach is a 422 carrying the driver's own message, and the target now says the attempt failed and when. `GET /v1/targets/{ref}/capabilities` answers from what is already known, without connecting. `noxis target probe` prints the same table.

## How will I solve it?

One `internal/adapters/mysql` package behind `ports.Driver`. The DSN is built once, in that package, from the driver's own `Config` — nothing else may assemble one. TLS modes map one to one onto the driver's names. The probe runs a single query that touches no table, `SELECT @@version, @@version_comment, @@max_allowed_packet`; `domain.ParseServerVersion` reads the flavour from the version string and comment, and `domain.CapabilitiesFor` derives the capability set from the measured server, adding a note when the declared driver disagrees with what answered. The `probed` outcome lands on the target row and in the journal.

## Done when

- [x] A probe distinguishes MySQL 8.4 from MariaDB 11.4 by what the server reports, verified against real containers
- [x] Capabilities are derived from the engine, so a route cannot offer what it cannot do
- [ ] `@slow` UAT runs both engines in containers — verified by hand this round, the harness is still `@fast`
- [x] The probe timeout comes from `PROBE_TIMEOUT`, and an unreachable target answers 422 rather than holding the request

## Risks & notes

- Both driver names map to one implementation on purpose: the engine is measured, not assumed, and a mismatch is a note rather than a refusal.
- A refused connection quotes the driver, with the password scrubbed out of the message in case a DSN is echoed back.
- Sealing is bound to the target `ref`; a password that cannot be opened is a 422 naming `SECRET_KEY` rather than a silent retry with an empty password.
