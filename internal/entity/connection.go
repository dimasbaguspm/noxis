package entity

import (
	"errors"
	"strings"

	"noxis/internal/constants"
)

// Connection is a registered SQL database. noxis connects to databases that
// already exist; it does not create them.
//
// ponytail: the password is stored as-is (single-operator trust model) and
// never leaves the API; encrypt at rest when this box stops being yours.
type Connection struct {
	Name      string            `json:"name" validate:"required"`
	Driver    string            `json:"driver" validate:"required"` // postgres | mysql (mariadb aliases mysql)
	Host      string            `json:"host" validate:"required"`
	Port      int               `json:"port" validate:"required"`
	User      string            `json:"user"`
	Password  string            `json:"password,omitempty"`
	Database  string            `json:"database" validate:"required"`
	Params    map[string]string `json:"params,omitempty"` // extra DSN params (sslmode, ...)
	CreatedAt int64             `json:"createdAt"`
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
	case constants.DriverRedis, constants.DriverValkey:
		return true
	}
	return false
}

// Redacted blanks the password for serving.
func (c Connection) Redacted() Connection {
	c.Password = ""
	return c
}
