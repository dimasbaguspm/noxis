package entity

import (
	"errors"
	"strings"
)

// Connection is a registered database instance. noxis connects to databases
// that already exist; it does not create them.
//
// The password never leaves the API. It cannot be hashed — it must be sent to
// the database — so it is stored as-is unless NOXIS_SECRET_KEY is set, in which
// case the store encrypts it at rest with AES-256-GCM.
type Connection struct {
	// ID is the stable instance identifier used by the cross-family
	// /v1/instances routes. Grants and changes still link by Name.
	ID string `json:"id,omitempty"`
	// Family names the engine family that serves the driver. It is not
	// stored: the cross-family routes set it when serving.
	Family    string            `json:"family,omitempty"`
	Name      string            `json:"name" validate:"required"`
	Driver    string            `json:"driver" validate:"required"` // postgres | mysql (mariadb aliases mysql)
	Host      string            `json:"host" validate:"required"`
	Port      int               `json:"port" validate:"required"`
	User      string            `json:"user"`
	Password  string            `json:"password,omitempty"`
	Database  string            `json:"database" validate:"required"`
	Params    map[string]string `json:"params,omitempty"` // extra DSN params (sslmode, ...)
	CreatedAt int64             `json:"createdAt"`
	CreatedBy string            `json:"createdBy,omitempty"`
}

// Validate checks the required fields. A user is required for SQL servers;
// Redis and Valkey commonly run without an ACL user, where only the password
// (or nothing) is set.
func (c Connection) Validate() error {
	if err := validate.Struct(c); err != nil {
		return err
	}
	if c.User == "" && !isKeyValueDriver(c.Driver) {
		return errors.New("user is required")
	}
	return nil
}

func isKeyValueDriver(driver string) bool {
	switch strings.ToLower(driver) {
	case "redis", "valkey": // memcache-family drivers need no user
		return true
	}
	return false
}

// Redacted blanks the password for serving.
func (c Connection) Redacted() Connection {
	c.Password = ""
	return c
}
