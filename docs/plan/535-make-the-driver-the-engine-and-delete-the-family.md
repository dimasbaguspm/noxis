# [engine] make the driver the engine and delete the family

- **Plan id:** issue 535
- **Area:** Engine
- **Status:** completed 2026-10-30
- **Issue:** [#535](https://github.com/dimasbaguspm/noxis/issues/535)
- **Pull requests:** [#542](https://github.com/dimasbaguspm/noxis/pull/542) (#536) · [#543](https://github.com/dimasbaguspm/noxis/pull/543) (#537) · [#544](https://github.com/dimasbaguspm/noxis/pull/544) (#538) · [#545](https://github.com/dimasbaguspm/noxis/pull/545) (#539) · [#546](https://github.com/dimasbaguspm/noxis/pull/546) (#540) · [#547](https://github.com/dimasbaguspm/noxis/pull/547) (#541)

## Plan

## What problem does this solve?

The console grouped every database under three engines — sql, memcache, mongodb — so a driver was a field inside one module rather than a module of its own. The API namespace, the change flow, the interfaces, the auth check and the console routes all keyed on that family, so one driver could not carry behavior another must not have, and a new family meant editing the same six files.

The cost showed up as a fat interface: every SQL driver was declared able to prove a change, when sqlite has no scratch schema, cockroach skips scratch and mysql commits DDL implicitly. The gap was handled at runtime, per request, by whoever remembered it.

## What is the user facing?

The engine path is the driver: `/external/v1/engine/postgres/...`, `/engine/mariadb/...`, `/engine/valkey/...`. There is no `sql` or `memcache` namespace left.

The driver picker lists every driver as its own choice, and the console routes drop the engine segment so a page is about the connection you picked, not about a family.

A driver that cannot do something says so once, plainly, instead of offering a button that fails later.

## How will I solve it?

One package per driver under the modules layer, each self-contained: postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongodb. They share nothing but the interfaces and the constants, by decision — a fix to one proof rule is applied to the others by hand, and a conformance suite outside the modules layer fails when a driver drifts from the contract.

The registry became a driver-to-module map. The fat interfaces split into capabilities, so a driver implements only what it can prove and the loader returns unsupported for the rest.

The change flow belongs to the driver. Routing a change to its queue stopped being a kind switch and became a lookup of the connection's driver.

## Done when

- [x] no family concept remains in the code, the wire or the console
- [x] every driver name is its own module package with its own verification rules
- [x] the engine path, the driver catalog and the change queues are keyed by driver
- [x] a capability a driver does not implement is refused at load with a named error, never stubbed
- [x] a conformance suite runs the same contract against every driver
- [x] lint, unit tests and the acceptance matrix pass; the OpenAPI documents the driver paths
- [x] adding a family is a new module directory plus one registry line, proven by the follow-up

## Risks & notes

This was a deliberate wire break: the engine path, the console routes and the family field on the instance payload all changed. No backward compatibility was required by decision — the store needed no migration, because a connection already records only its driver and the family was never persisted.

Duplication is accepted, not accidental: roughly 2.5k lines of SQL machinery landed in five packages. The conformance suite and the acceptance matrix are the only guards, so they were not optional.

Extends #520, which made a driver's behavior declarative; this epic made the driver the unit everything else is keyed on.

## Tickets

| Ticket | Slice | Pull request |
| --- | --- | --- |
| #536 | registry, wire and access gate keyed on the driver; family deleted | #542 |
| #537 | module interfaces split into capabilities, mocks regenerated | #543 |
| #538 | one package per SQL driver, conformance suite lands with the split | #544 |
| #539 | redis, valkey and mongodb as their own packages | #545 |
| #540 | console routes drop the engine segment, navigation from the driver catalog | #546 |
| #541 | agent brief, architecture doc, README, OpenAPI artifacts and this snapshot | #547 |

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
