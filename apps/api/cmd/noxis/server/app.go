package server

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	authsvc "noxis/internal/auth"
	"noxis/internal/common/metrics"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/modules"
	"noxis/internal/repositories"
	"noxis/internal/usecases"
	"noxis/internal/usecases/access"
	"noxis/internal/usecases/apply"
	"noxis/internal/usecases/auth"
	"noxis/internal/usecases/cache"
	"noxis/internal/usecases/change"
	"noxis/internal/usecases/connection"
	"noxis/internal/usecases/driver"
	"noxis/internal/usecases/instance"
	"noxis/internal/usecases/metadata"
	"noxis/internal/usecases/mongo"
	"noxis/internal/usecases/permission"
	"noxis/internal/usecases/schema"
	"noxis/internal/usecases/user"
)

// sources are the seams the composition root builds the use cases from. They are interfaces, so a test can hand in fakes without a store or an HTTP stack.
type sources struct {
	Repositories repositories.Registry
	Modules      modules.Registry
	Auth         entity.AuthConfig
	AuthService  *authsvc.Service
	AccessTTL    time.Duration
}

// app is the composition root: every use case built once, explicitly.
type app struct {
	Modules modules.Registry
	Auth    entity.AuthConfig

	ConnectionList   connection.List
	Connection       connection.Get
	SaveConnection   connection.Save
	DeleteConnection connection.Delete
	TestConnection   connection.Test

	Schema schema.Tables
	ERD    schema.ERD
	Rows   schema.Rows
	Op     schema.Op

	Members          permission.Members
	AccessMatrix     permission.Matrix
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

	UserList   user.List
	UserGet    user.Get
	UserUpdate user.Update

	AccessCreate  access.Create
	AccessList    access.List
	AccessApprove access.Approve
	AccessReject  access.Reject

	Drivers   driver.Drivers
	Levels    metadata.Levels
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

// newApp builds every use case from s.
func newApp(s sources) app {
	// Instruments come from the global meter provider the logger setup registered; without a collector every instrument is a no-op.
	m, err := metrics.New(otel.Meter(metrics.Scope))
	if err != nil {
		slog.Error("metric instruments", "error", err)
	}

	// The registry owns the engine families and their change queues, so it is built here where the instruments it reports through exist.
	if s.Modules == nil {
		s.Modules = modules.Default(s.Repositories, m)
	}

	deps := usecases.RegistryDeps{
		Repositories: s.Repositories,
		Modules:      s.Modules,
		Auth:         s.AuthService,
		AccessTTL:    s.AccessTTL,
		Metrics:      m,
	}
	registerConnectionGauge(m, s.Repositories.Connections)
	return app{
		Modules: s.Modules,
		Auth:    s.Auth,

		ConnectionList:   connection.List{RegistryDeps: deps},
		Connection:       connection.Get{RegistryDeps: deps},
		SaveConnection:   connection.Save{RegistryDeps: deps},
		DeleteConnection: connection.Delete{RegistryDeps: deps},
		TestConnection:   connection.Test{RegistryDeps: deps},

		Schema: schema.Tables{RegistryDeps: deps},
		ERD:    schema.ERD{RegistryDeps: deps},
		Rows:   schema.Rows{RegistryDeps: deps},
		Op:     schema.Op{RegistryDeps: deps},

		Members:          permission.Members{RegistryDeps: deps},
		AccessMatrix:     permission.Matrix{RegistryDeps: deps},
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

		UserList:   user.List{RegistryDeps: deps},
		UserGet:    user.Get{RegistryDeps: deps},
		UserUpdate: user.Update{RegistryDeps: deps},

		AccessCreate:  access.Create{RegistryDeps: deps},
		AccessList:    access.List{RegistryDeps: deps},
		AccessApprove: access.Approve{RegistryDeps: deps},
		AccessReject:  access.Reject{RegistryDeps: deps},

		Drivers:   driver.Drivers{RegistryDeps: deps},
		Levels:    metadata.Levels{RegistryDeps: deps},
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

// registerConnectionGauge reports connections per family and driver from the repository on the collector's cadence.
func registerConnectionGauge(m *metrics.Metrics, connections interfaces.ConnectionRepository) {
	if m == nil {
		return
	}
	_, err := otel.Meter(metrics.Scope).RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		list, err := connections.List(ctx)
		if err != nil {
			return err
		}
		counts := map[[2]string]int64{}
		for i := range list {
			counts[[2]string{list[i].Family, list[i].Driver}]++
		}
		for key, n := range counts {
			o.ObserveInt64(m.Connections, n, metric.WithAttributes(
				attribute.String(constants.LabelFamily, key[0]),
				attribute.String(constants.LabelDriver, key[1]),
			))
		}
		return nil
	}, m.Connections)
	if err != nil {
		slog.Error("connection gauge", "error", err)
	}
}
