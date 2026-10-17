package server

import (
	"noxis/internal/common/config"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
	"noxis/internal/usecases/apply"
	"noxis/internal/usecases/auth"
	"noxis/internal/usecases/cache"
	"noxis/internal/usecases/change"
	"noxis/internal/usecases/connection"
	"noxis/internal/usecases/driver"
	"noxis/internal/usecases/instance"
	"noxis/internal/usecases/mongo"
	"noxis/internal/usecases/permission"
	"noxis/internal/usecases/schema"
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

// app is the composition root: every use case built once, explicitly.
type app struct {
	Modules modules.Registry
	Auth    config.AuthInfo

	ConnectionList   connection.List
	Connection       connection.Get
	SaveConnection   connection.Save
	DeleteConnection connection.Delete
	TestConnection   connection.Test

	Schema schema.Tables
	Rows   schema.Rows
	Op     schema.Op

	Members          permission.Members
	ListPermissions  permission.List
	SetPermission    permission.Set
	DeletePermission permission.Delete

	Instances      instance.List
	Instance       instance.Get
	RotatePassword instance.RotatePassword

	AuthConfig auth.Config
	Login      auth.Login
	Logout     auth.Logout
	ChangePass auth.Password
	Me         auth.Me

	Drivers   driver.Drivers
	CacheKeys cache.Keys
	CacheVal  cache.Value
	CacheDel  cache.Delete
	CacheFlsh cache.Flush

	MongoDbs   mongo.Databases
	MongoColls mongo.Collections
	MongoRun   mongo.Run

	ChangeList     change.List
	ChangeSubmit   change.Submit
	ChangeGet      change.Get
	ChangeApprove  change.Approve
	ChangeReject   change.Reject
	ChangeSchedule change.Schedule

	ChangeApply apply.Apply
	Tick        apply.Tick
}

// newApp builds every use case from s. It is the single place the use cases
// are constructed.
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

		ConnectionList:   connection.List{RegistryDeps: deps},
		Connection:       connection.Get{RegistryDeps: deps},
		SaveConnection:   connection.Save{RegistryDeps: deps},
		DeleteConnection: connection.Delete{RegistryDeps: deps},
		TestConnection:   connection.Test{RegistryDeps: deps},

		Schema: schema.Tables{RegistryDeps: deps},
		Rows:   schema.Rows{RegistryDeps: deps},
		Op:     schema.Op{RegistryDeps: deps},

		Members:          permission.Members{RegistryDeps: deps},
		ListPermissions:  permission.List{RegistryDeps: deps},
		SetPermission:    permission.Set{RegistryDeps: deps},
		DeletePermission: permission.Delete{RegistryDeps: deps},

		Instances:      instance.List{RegistryDeps: deps},
		Instance:       instance.Get{RegistryDeps: deps},
		RotatePassword: instance.RotatePassword{RegistryDeps: deps},

		AuthConfig: auth.Config{RegistryDeps: deps, Auth: s.Auth},
		Login:      auth.Login{RegistryDeps: deps},
		Logout:     auth.Logout{RegistryDeps: deps},
		ChangePass: auth.Password{RegistryDeps: deps},
		Me:         auth.Me{RegistryDeps: deps},

		Drivers:   driver.Drivers{RegistryDeps: deps},
		CacheKeys: cache.Keys{RegistryDeps: deps},
		CacheVal:  cache.Value{RegistryDeps: deps},
		CacheDel:  cache.Delete{RegistryDeps: deps},
		CacheFlsh: cache.Flush{RegistryDeps: deps},

		MongoDbs:   mongo.Databases{RegistryDeps: deps},
		MongoColls: mongo.Collections{RegistryDeps: deps},
		MongoRun:   mongo.Run{RegistryDeps: deps},

		ChangeList:     change.List{RegistryDeps: deps},
		ChangeSubmit:   change.Submit{RegistryDeps: deps},
		ChangeGet:      change.Get{RegistryDeps: deps},
		ChangeApprove:  change.Approve{RegistryDeps: deps},
		ChangeReject:   change.Reject{RegistryDeps: deps},
		ChangeSchedule: change.Schedule{RegistryDeps: deps},

		ChangeApply: apply.Apply{RegistryDeps: deps},
		Tick:        apply.Tick{RegistryDeps: deps},
	}
}
