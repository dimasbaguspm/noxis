package deps

import (
	"noxis/internal/usecases"
)

// Init binds every use case once, from the sources already set on d. It is
// the single place the use cases are constructed.
func Init(d Deps) Deps {
	d.SubmitChange = WithDeps(d, usecases.SubmitChange{})
	d.Approve = WithDeps(d, usecases.Approve{})
	d.Schedule = WithDeps(d, usecases.Schedule{})
	d.Apply = WithDeps(d, usecases.Apply{})
	d.Reject = WithDeps(d, usecases.Reject{})
	d.SQLChanges = WithDeps(d, usecases.SQLChanges{})
	d.SQLChange = WithDeps(d, usecases.SQLChange{})
	d.Tick = WithDeps(d, usecases.Tick{})
	d.ConnectionList = WithDeps(d, usecases.ConnectionList{})
	d.Connection = WithDeps(d, usecases.Connection{})
	d.Raw = WithDeps(d, usecases.Raw{})
	d.SaveConnection = WithDeps(d, usecases.SaveConnection{})
	d.DeleteConnection = WithDeps(d, usecases.DeleteConnection{})
	d.TestConnection = WithDeps(d, usecases.TestConnection{})
	d.Schema = WithDeps(d, usecases.Schema{})
	d.Rows = WithDeps(d, usecases.Rows{})
	d.Op = WithDeps(d, usecases.Op{})
	d.SubmitMongo = WithDeps(d, usecases.SubmitMongoChange{})
	d.MongoChange = WithDeps(d, usecases.MongoChange{})
	d.ApplyMongo = WithDeps(d, usecases.ApplyMongo{})
	d.MongoChanges = WithDeps(d, usecases.MongoChanges{})
	d.Instance = WithDeps(d, usecases.Instance{})
	d.Instances = WithDeps(d, usecases.ListInstances{})
	d.RotatePassword = WithDeps(d, usecases.RotatePassword{})
	d.ListGrants = WithDeps(d, usecases.ListGrants{})
	d.SetGrant = WithDeps(d, usecases.SetGrant{})
	d.DeleteGrant = WithDeps(d, usecases.DeleteGrant{})
	d.Groups = WithDeps(d, usecases.Groups{})
	d.ConnectionLevels = WithDeps(d, usecases.ConnectionLevels{})
	return d
}
