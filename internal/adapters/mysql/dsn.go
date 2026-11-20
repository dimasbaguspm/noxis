// Package mysql is the one engine adapter: it speaks to MySQL and MariaDB through the same
// wire protocol, and reads the differences from the server instead of assuming them from the
// name a target was declared with.
package mysql

import (
	"net"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strconv"

	mysqldriver "github.com/go-sql-driver/mysql"
)

// tlsSetting maps a target's declared mode onto the driver's own option, one to one. Nothing
// is upgraded silently: `preferred` still encrypts when the server offers it, and
// `skip-verify` never claims to have verified a certificate.
func tlsSetting(mode string) string {
	switch mode {
	case constants.TLSVerify:
		return "true"
	case constants.TLSSkipVerify:
		return "skip-verify"
	default:
		return "preferred"
	}
}

// dsnOf builds the only connection string this build uses. It stays in this package so no
// handler can assemble one, and so a password never ends up in a URL somebody logs.
func dsnOf(target domain.Target, password string) string {
	cfg := mysqldriver.NewConfig()
	cfg.User = target.Username
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	cfg.DBName = target.Database
	cfg.ParseTime = true
	cfg.TLSConfig = tlsSetting(target.TLSMode)
	cfg.AllowNativePasswords = true
	if len(target.Options) > 0 {
		cfg.Params = target.Options
	}
	return cfg.FormatDSN()
}
