package entity

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
	User      string            `json:"user" validate:"required"`
	Password  string            `json:"password,omitempty"`
	Database  string            `json:"database" validate:"required"`
	Params    map[string]string `json:"params,omitempty"` // extra DSN params (sslmode, ...)
	CreatedAt int64             `json:"createdAt"`
}

// Validate checks the required fields.
func (c Connection) Validate() error { return validate.Struct(c) }

// Redacted blanks the password for serving.
func (c Connection) Redacted() Connection {
	c.Password = ""
	return c
}
