package deps

import (
	"noxis/internal/common/config"
	"noxis/internal/modules"
	"noxis/internal/usecases"
)

// Uses is what a request can call: the use cases bound to the boot-time
// sources, plus the two transport bits a route needs (the engine registry for
// {engine} resolution and the public OIDC config). Init builds it once.
type Uses struct {
	Modules modules.Registry
	Auth    config.AuthInfo

	SubmitChange     usecases.SubmitChange
	Approve          usecases.Approve
	Schedule         usecases.Schedule
	Apply            usecases.Apply
	Reject           usecases.Reject
	SQLChanges       usecases.SQLChanges
	SQLChange        usecases.SQLChange
	Tick             usecases.Tick
	ConnectionList   usecases.ConnectionList
	Connection       usecases.Connection
	Raw              usecases.Raw
	SaveConnection   usecases.SaveConnection
	DeleteConnection usecases.DeleteConnection
	TestConnection   usecases.TestConnection
	Schema           usecases.Schema
	Rows             usecases.Rows
	Op               usecases.Op
	SubmitMongo      usecases.SubmitMongoChange
	MongoChange      usecases.MongoChange
	ApplyMongo       usecases.ApplyMongo
	MongoChanges     usecases.MongoChanges
	Instance         usecases.Instance
	Instances        usecases.ListInstances
	RotatePassword   usecases.RotatePassword
	ListGrants       usecases.ListGrants
	SetGrant         usecases.SetGrant
	DeleteGrant      usecases.DeleteGrant
	Groups           usecases.Groups
	ConnectionLevels usecases.ConnectionLevels
}

// Init binds every use case once, from the sources in d. It is the single
// place the use cases are constructed.
func Init(d Deps) Uses {
	return Uses{
		Modules: d.Modules,
		Auth:    d.Auth,

		SubmitChange:     WithDeps(d, usecases.SubmitChange{}),
		Approve:          WithDeps(d, usecases.Approve{}),
		Schedule:         WithDeps(d, usecases.Schedule{}),
		Apply:            WithDeps(d, usecases.Apply{}),
		Reject:           WithDeps(d, usecases.Reject{}),
		SQLChanges:       WithDeps(d, usecases.SQLChanges{}),
		SQLChange:        WithDeps(d, usecases.SQLChange{}),
		Tick:             WithDeps(d, usecases.Tick{}),
		ConnectionList:   WithDeps(d, usecases.ConnectionList{}),
		Connection:       WithDeps(d, usecases.Connection{}),
		Raw:              WithDeps(d, usecases.Raw{}),
		SaveConnection:   WithDeps(d, usecases.SaveConnection{}),
		DeleteConnection: WithDeps(d, usecases.DeleteConnection{}),
		TestConnection:   WithDeps(d, usecases.TestConnection{}),
		Schema:           WithDeps(d, usecases.Schema{}),
		Rows:             WithDeps(d, usecases.Rows{}),
		Op:               WithDeps(d, usecases.Op{}),
		SubmitMongo:      WithDeps(d, usecases.SubmitMongoChange{}),
		MongoChange:      WithDeps(d, usecases.MongoChange{}),
		ApplyMongo:       WithDeps(d, usecases.ApplyMongo{}),
		MongoChanges:     WithDeps(d, usecases.MongoChanges{}),
		Instance:         WithDeps(d, usecases.Instance{}),
		Instances:        WithDeps(d, usecases.ListInstances{}),
		RotatePassword:   WithDeps(d, usecases.RotatePassword{}),
		ListGrants:       WithDeps(d, usecases.ListGrants{}),
		SetGrant:         WithDeps(d, usecases.SetGrant{}),
		DeleteGrant:      WithDeps(d, usecases.DeleteGrant{}),
		Groups:           WithDeps(d, usecases.Groups{}),
		ConnectionLevels: WithDeps(d, usecases.ConnectionLevels{}),
	}
}
