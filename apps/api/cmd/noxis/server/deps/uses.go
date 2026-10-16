package deps

import (
	"noxis/internal/common/config"
	"noxis/internal/modules"
	"noxis/internal/usecases"
)

// Uses is what a request can call: the use cases bound to the boot-time
// sources, plus the transport bits a route needs (the registry for {engine}
// resolution and the auth service, and the public auth config). Init builds it
// once.
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
	ListPermissions  usecases.ListPermissions
	SetPermission    usecases.SetPermission
	DeletePermission usecases.DeletePermission
	Members          usecases.Members
	ConnectionLevels usecases.ConnectionLevels

	AuthConfig     usecases.AuthConfig
	Login          usecases.Login
	Logout         usecases.Logout
	ChangePass     usecases.ChangePassword
	Drivers        usecases.Drivers
	CacheKeys      usecases.CacheKeys
	CacheValue     usecases.CacheValue
	CacheDelete    usecases.CacheDelete
	CacheFlush     usecases.CacheFlush
	MongoDbs       usecases.MongoDatabases
	MongoColls     usecases.MongoCollections
	MongoRun       usecases.MongoRun
	ChangeList     usecases.ChangeList
	ChangeSubmit   usecases.ChangeSubmit
	ChangeGet      usecases.ChangeGet
	ChangeApprove  usecases.ChangeApprove
	ChangeApply    usecases.ChangeApply
	ChangeReject   usecases.ChangeReject
	ChangeSchedule usecases.ChangeSchedule
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
		ListPermissions:  WithDeps(d, usecases.ListPermissions{}),
		SetPermission:    WithDeps(d, usecases.SetPermission{}),
		DeletePermission: WithDeps(d, usecases.DeletePermission{}),
		Members:          WithDeps(d, usecases.Members{}),
		ConnectionLevels: WithDeps(d, usecases.ConnectionLevels{}),

		AuthConfig:     WithDeps(d, usecases.AuthConfig{}),
		Login:          WithDeps(d, usecases.Login{}),
		Logout:         WithDeps(d, usecases.Logout{}),
		ChangePass:     WithDeps(d, usecases.ChangePassword{}),
		Drivers:        WithDeps(d, usecases.Drivers{}),
		CacheKeys:      WithDeps(d, usecases.CacheKeys{}),
		CacheValue:     WithDeps(d, usecases.CacheValue{}),
		CacheDelete:    WithDeps(d, usecases.CacheDelete{}),
		CacheFlush:     WithDeps(d, usecases.CacheFlush{}),
		MongoDbs:       WithDeps(d, usecases.MongoDatabases{}),
		MongoColls:     WithDeps(d, usecases.MongoCollections{}),
		MongoRun:       WithDeps(d, usecases.MongoRun{}),
		ChangeList:     WithDeps(d, usecases.ChangeList{}),
		ChangeSubmit:   WithDeps(d, usecases.ChangeSubmit{}),
		ChangeGet:      WithDeps(d, usecases.ChangeGet{}),
		ChangeApprove:  WithDeps(d, usecases.ChangeApprove{}),
		ChangeApply:    WithDeps(d, usecases.ChangeApply{}),
		ChangeReject:   WithDeps(d, usecases.ChangeReject{}),
		ChangeSchedule: WithDeps(d, usecases.ChangeSchedule{}),
	}
}
