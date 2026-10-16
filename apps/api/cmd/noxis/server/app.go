package server

import (
	"noxis/internal/common/config"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
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

// app is the composition root: every use case built once, explicitly, with the
// fields it needs named at the call site. There is no reflection and no
// dependency container in a request context.
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

// newApp builds every use case from s. It is the single place the use cases
// are constructed.
func newApp(s sources) app {
	reg := usecases.RegistryDeps{Modules: s.Modules}
	return app{
		Modules: s.Modules,
		Auth:    s.Auth,

		SubmitChange:     usecases.SubmitChange{RegistryDeps: reg, Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		Approve:          usecases.Approve{Changes: s.Changes, Permissions: s.Permissions},
		Schedule:         usecases.Schedule{Changes: s.Changes, Permissions: s.Permissions},
		Apply:            usecases.Apply{RegistryDeps: reg, Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		Reject:           usecases.Reject{Changes: s.Changes, Permissions: s.Permissions},
		SQLChanges:       usecases.SQLChanges{Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		SQLChange:        usecases.SQLChange{Changes: s.Changes, Permissions: s.Permissions},
		Tick:             usecases.Tick{RegistryDeps: reg, Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		ConnectionList:   usecases.ConnectionList{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		Connection:       usecases.Connection{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		Raw:              usecases.Raw{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		SaveConnection:   usecases.SaveConnection{RegistryDeps: reg, Connections: s.Connections},
		DeleteConnection: usecases.DeleteConnection{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		TestConnection:   usecases.TestConnection{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		Schema:           usecases.Schema{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		Rows:             usecases.Rows{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		Op:               usecases.Op{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		SubmitMongo:      usecases.SubmitMongoChange{RegistryDeps: reg, Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		MongoChange:      usecases.MongoChange{Changes: s.Changes, Permissions: s.Permissions},
		ApplyMongo:       usecases.ApplyMongo{RegistryDeps: reg, Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		MongoChanges:     usecases.MongoChanges{Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		Instance:         usecases.Instance{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		Instances:        usecases.ListInstances{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		RotatePassword:   usecases.RotatePassword{RegistryDeps: reg, Connections: s.Connections},
		ListPermissions:  usecases.ListPermissions{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		SetPermission:    usecases.SetPermission{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		DeletePermission: usecases.DeletePermission{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		Members:          usecases.Members{Permissions: s.Permissions, Users: s.Users},
		ConnectionLevels: usecases.ConnectionLevels{Connections: s.Connections, Permissions: s.Permissions},
		AuthConfig:       usecases.AuthConfig{Auth: s.Auth},
		Login:            usecases.Login{RegistryDeps: reg},
		Logout:           usecases.Logout{RegistryDeps: reg},
		ChangePass:       usecases.ChangePassword{RegistryDeps: reg},
		Drivers:          usecases.Drivers{},
		CacheKeys:        usecases.CacheKeys{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		CacheValue:       usecases.CacheValue{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		CacheDelete:      usecases.CacheDelete{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		CacheFlush:       usecases.CacheFlush{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		MongoDbs:         usecases.MongoDatabases{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		MongoColls:       usecases.MongoCollections{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		MongoRun:         usecases.MongoRun{RegistryDeps: reg, Connections: s.Connections, Permissions: s.Permissions},
		ChangeList:       usecases.ChangeList{RegistryDeps: reg, Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		ChangeSubmit:     usecases.ChangeSubmit{RegistryDeps: reg, Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		ChangeGet:        usecases.ChangeGet{RegistryDeps: reg, Changes: s.Changes, Permissions: s.Permissions},
		ChangeApprove:    usecases.ChangeApprove{RegistryDeps: reg, Changes: s.Changes, Permissions: s.Permissions},
		ChangeApply:      usecases.ChangeApply{RegistryDeps: reg, Connections: s.Connections, Changes: s.Changes, Permissions: s.Permissions},
		ChangeReject:     usecases.ChangeReject{RegistryDeps: reg, Changes: s.Changes, Permissions: s.Permissions},
		ChangeSchedule:   usecases.ChangeSchedule{RegistryDeps: reg, Changes: s.Changes, Permissions: s.Permissions},
	}
}
