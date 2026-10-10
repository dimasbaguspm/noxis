package sql

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"

	// database/sql drivers, registered for the drivers we accept.
	_ "github.com/jackc/pgx/v5/stdlib"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// sqlDriver maps a driver name to a database/sql driver.
func sqlDriver(driver string) (string, error) {
	switch strings.ToLower(driver) {
	case constants.DriverPostgres, "postgresql":
		return "pgx", nil
	case constants.DriverMySQL, "mariadb":
		return "mysql", nil
	}
	return "", fmt.Errorf("unsupported driver %q (postgres or mysql)", driver)
}

// isMySQL reports whether the driver speaks the mysql protocol.
func isMySQL(driver string) bool {
	switch strings.ToLower(driver) {
	case constants.DriverMySQL, "mariadb":
		return true
	}
	return false
}

// dsn builds the driver-native connection string.
func dsn(c entity.Connection) (string, error) {
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	if isMySQL(c.Driver) {
		mc := mysql.NewConfig()
		mc.User, mc.Passwd = c.User, c.Password
		mc.Net, mc.Addr = "tcp", addr
		mc.DBName = c.Database
		mc.ParseTime = true
		mc.Params = c.Params
		return mc.FormatDSN(), nil
	}
	if _, err := sqlDriver(c.Driver); err != nil {
		return "", err
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   addr,
		Path:   "/" + c.Database,
	}
	q := u.Query()
	q.Set("sslmode", "disable") // homelab default; override via params
	for k, v := range c.Params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
