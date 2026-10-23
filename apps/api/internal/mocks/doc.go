// Package mocks holds the generated test doubles for the seams a use case
// depends on: the repositories (persistence) and the engine modules (external
// services). A use-case test declares only the calls it cares about instead of
// sharing a hand-written fake.
//
// Regenerate with `go generate ./internal/mocks` after an interface in
// internal/interfaces changes.
//
//go:generate go run go.uber.org/mock/mockgen -package mocks -destination repositories.go noxis/internal/interfaces ConnectionRepository,ChangeRepository,PermissionRepository,UserRepository,AccessEventRepository,AccessRequestRepository
//go:generate go run go.uber.org/mock/mockgen -package mocks -destination modules.go noxis/internal/interfaces SQLModule,MemcacheModule,MongoModule
package mocks
