package entity

import (
	"errors"
	"strings"
)

// Connection is a registered database instance. noxis connects to databases that already exist; it does not create them. The password never leaves the API.
type Connection struct {
	// ID identifies an instance across drivers; permissions and changes still link by Name.
	ID        string            `json:"id,omitempty"`
	Name      string            `json:"name" validate:"required"`
	Driver    string            `json:"driver" validate:"required"` // postgres | mysql (mariadb aliases mysql) | sqlite
	Host      string            `json:"host"`
	Port      int               `json:"port"`
	User      string            `json:"user"`
	Password  string            `json:"password,omitempty"`
	Database  string            `json:"database" validate:"required"`
	Params    map[string]string `json:"params,omitempty"` // extra DSN params (sslmode, ...)
	CreatedAt int64             `json:"createdAt"`
	CreatedBy string            `json:"createdBy,omitempty"`
}

// Validate checks the required fields. A user is required for SQL servers; Redis and Valkey commonly run without an ACL user, where only the password (or nothing) is set.
func (c Connection) Validate() error {
	if err := validate.Struct(c); err != nil {
		return err
	}
	if isFileDriver(c.Driver) {
		return nil
	}
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Port == 0 {
		return errors.New("port is required")
	}
	if c.User == "" && !allowsAnonymous(c.Driver) {
		return errors.New("user is required")
	}
	return nil
}

func isFileDriver(driver string) bool {
	return strings.EqualFold(driver, "sqlite") // the file path is Database
}

// allowsAnonymous reports the drivers that commonly run without an ACL user: a cache server with no ACL, and a search
// cluster with security disabled. Every other driver needs a user, so a typo in the field is caught at registration.
func allowsAnonymous(driver string) bool {
	switch strings.ToLower(driver) {
	case "redis", "valkey", "dragonfly", "elasticsearch", "meilisearch", "splunk":
		return true
	}
	return false
}

// Redacted blanks the password for serving.
func (c Connection) Redacted() Connection {
	c.Password = ""
	return c
}
