package server

import (
	"noxis/internal/common/config"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
)

// sources are the seams the composition root builds the use cases from. They
// are interfaces, so a test can hand in fakes without a store or an HTTP stack.
type sources struct {
	Connections interfaces.ConnectionRepository
	Changes     interfaces.ChangeRepository
	Permissions interfaces.PermissionRepository
	Users       interfaces.UserRepository
	Modules     modules.Registry
	Auth        config.AuthInfo
}

// app is the composition root: every use case built once, explicitly. There is
// no reflection and no dependency container in a request context.
type app struct {
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
	AuthConfig       usecases.AuthConfig
	Login            usecases.Login
	Logout           usecases.Logout
	ChangePass       usecases.ChangePassword
	Drivers          usecases.Drivers
	CacheKeys        usecases.CacheKeys
	CacheValue       usecases.CacheValue
	CacheDelete      usecases.CacheDelete
	CacheFlush       usecases.CacheFlush
	MongoDbs         usecases.MongoDatabases
	MongoColls       usecases.MongoCollections
	MongoRun         usecases.MongoRun
	ChangeList       usecases.ChangeList
	ChangeSubmit     usecases.ChangeSubmit
	ChangeGet        usecases.ChangeGet
	ChangeApprove    usecases.ChangeApprove
	ChangeApply      usecases.ChangeApply
	ChangeReject     usecases.ChangeReject
	ChangeSchedule   usecases.ChangeSchedule
}

// newApp builds every use case from s: one shared RegistryDeps value carries
// the repositories and the module registry, so each use case is a one-field
// literal. It is the single place the use cases are constructed.
func newApp(s sources) app {
	deps := usecases.RegistryDeps{
		Registry: repositories.Registry{
			Connections: s.Connections,
			Changes:     s.Changes,
			Permissions: s.Permissions,
			Users:       s.Users,
		},
		Modules: s.Modules,
	}
	return app{
		Modules: s.Modules,
		Auth:    s.Auth,

		SubmitChange:     usecases.SubmitChange{RegistryDeps: deps},
		Approve:          usecases.Approve{RegistryDeps: deps},
		Schedule:         usecases.Schedule{RegistryDeps: deps},
		Apply:            usecases.Apply{RegistryDeps: deps},
		Reject:           usecases.Reject{RegistryDeps: deps},
		SQLChanges:       usecases.SQLChanges{RegistryDeps: deps},
		SQLChange:        usecases.SQLChange{RegistryDeps: deps},
		Tick:             usecases.Tick{RegistryDeps: deps},
		ConnectionList:   usecases.ConnectionList{RegistryDeps: deps},
		Connection:       usecases.Connection{RegistryDeps: deps},
		Raw:              usecases.Raw{RegistryDeps: deps},
		SaveConnection:   usecases.SaveConnection{RegistryDeps: deps},
		DeleteConnection: usecases.DeleteConnection{RegistryDeps: deps},
		TestConnection:   usecases.TestConnection{RegistryDeps: deps},
		Schema:           usecases.Schema{RegistryDeps: deps},
		Rows:             usecases.Rows{RegistryDeps: deps},
		Op:               usecases.Op{RegistryDeps: deps},
		SubmitMongo:      usecases.SubmitMongoChange{RegistryDeps: deps},
		MongoChange:      usecases.MongoChange{RegistryDeps: deps},
		ApplyMongo:       usecases.ApplyMongo{RegistryDeps: deps},
		MongoChanges:     usecases.MongoChanges{RegistryDeps: deps},
		Instance:         usecases.Instance{RegistryDeps: deps},
		Instances:        usecases.ListInstances{RegistryDeps: deps},
		RotatePassword:   usecases.RotatePassword{RegistryDeps: deps},
		ListPermissions:  usecases.ListPermissions{RegistryDeps: deps},
		SetPermission:    usecases.SetPermission{RegistryDeps: deps},
		DeletePermission: usecases.DeletePermission{RegistryDeps: deps},
		Members:          usecases.Members{RegistryDeps: deps},
		ConnectionLevels: usecases.ConnectionLevels{RegistryDeps: deps},
		AuthConfig:       usecases.AuthConfig{Auth: s.Auth},
		Login:            usecases.Login{RegistryDeps: deps},
		Logout:           usecases.Logout{RegistryDeps: deps},
		ChangePass:       usecases.ChangePassword{RegistryDeps: deps},
		Drivers:          usecases.Drivers{},
		CacheKeys:        usecases.CacheKeys{RegistryDeps: deps},
		CacheValue:       usecases.CacheValue{RegistryDeps: deps},
		CacheDelete:      usecases.CacheDelete{RegistryDeps: deps},
		CacheFlush:       usecases.CacheFlush{RegistryDeps: deps},
		MongoDbs:         usecases.MongoDatabases{RegistryDeps: deps},
		MongoColls:       usecases.MongoCollections{RegistryDeps: deps},
		MongoRun:         usecases.MongoRun{RegistryDeps: deps},
		ChangeList:       usecases.ChangeList{RegistryDeps: deps},
		ChangeSubmit:     usecases.ChangeSubmit{RegistryDeps: deps},
		ChangeGet:        usecases.ChangeGet{RegistryDeps: deps},
		ChangeApprove:    usecases.ChangeApprove{RegistryDeps: deps},
		ChangeApply:      usecases.ChangeApply{RegistryDeps: deps},
		ChangeReject:     usecases.ChangeReject{RegistryDeps: deps},
		ChangeSchedule:   usecases.ChangeSchedule{RegistryDeps: deps},
	}
}
