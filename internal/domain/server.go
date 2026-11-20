package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"noxis/internal/constants"
)

// Server is what a probe learned about the engine behind a target. Flavor is measured, not
// the name the target was declared with: a target that says `mariadb` while the socket
// answers as MySQL is a fact the reviewer needs, not a typo to hide.
type Server struct {
	Flavor  string
	Version string
	Major   int
	Minor   int
	Patch   int
}

var versionNumber = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// ParseServerVersion reads what the two engines report. MariaDB names itself in the version
// string; MySQL does not, which is why the comment is read as well.
func ParseServerVersion(version, comment string) Server {
	server := Server{Version: strings.TrimSpace(version), Flavor: constants.DriverMySQL}
	if strings.Contains(strings.ToLower(fmt.Sprintf("%s %s", server.Version, comment)), "mariadb") {
		server.Flavor = constants.DriverMariaDB
	}
	if match := versionNumber.FindStringSubmatch(server.Version); match != nil {
		server.Major, _ = strconv.Atoi(match[1])
		server.Minor, _ = strconv.Atoi(match[2])
		server.Patch, _ = strconv.Atoi(match[3])
	}
	return server
}

// MeasuredServer rebuilds what a stored target knows about its engine: the measured flavour
// and version when the target has been probed, the declared driver when it has not.
func MeasuredServer(driver, version string) Server {
	if strings.TrimSpace(version) == "" {
		return Server{Flavor: driver}
	}
	return ParseServerVersion(version, "")
}

// AtLeast reports whether the measured server is at least major.minor.
func (s Server) AtLeast(major, minor int) bool {
	if s.Major != major {
		return s.Major > major
	}
	return s.Minor >= minor
}

// Probed reports whether anything is known about the engine yet.
func (s Server) Probed() bool { return s.Version != "" }

// Capabilities is what this build may offer for one engine. Every field is derived from the
// engine, never asserted by a route: a page that offers something the engine cannot do is a
// lie the reviewer only discovers after approving.
type Capabilities struct {
	Driver             string   `json:"driver"`
	Flavor             string   `json:"flavor"`
	Version            string   `json:"version"`
	Major              int      `json:"major"`
	Minor              int      `json:"minor"`
	Patch              int      `json:"patch"`
	TransactionalDML   bool     `json:"transactional_dml"`
	ShadowDDLRehearsal bool     `json:"shadow_ddl_rehearsal"`
	RowImageRehearsal  bool     `json:"row_image_rehearsal"`
	IndexMetadata      bool     `json:"index_metadata"`
	ForeignKeyMetadata bool     `json:"foreign_key_metadata"`
	ExplainJSON        bool     `json:"explain_json"`
	Notes              []string `json:"notes"`
}

// CapabilitiesFor derives the capability set from the engine. Both engines run InnoDB and
// both expose SHOW CREATE TABLE, information_schema and EXPLAIN FORMAT=JSON within the
// versions this build supports; what differs is the floor, and the notes say so.
func CapabilitiesFor(declared string, server Server) Capabilities {
	caps := Capabilities{
		Driver:             declared,
		Flavor:             server.Flavor,
		Version:            server.Version,
		Major:              server.Major,
		Minor:              server.Minor,
		Patch:              server.Patch,
		TransactionalDML:   true,
		ShadowDDLRehearsal: true,
		RowImageRehearsal:  true,
		IndexMetadata:      true,
		ForeignKeyMetadata: true,
	}
	switch server.Flavor {
	case constants.DriverMariaDB:
		caps.ExplainJSON = server.AtLeast(10, 1)
		if server.Probed() && !server.AtLeast(10, 5) {
			caps.Notes = append(caps.Notes, "before MariaDB 10.5 there is no instant DDL: a rehearsal copies the table")
		}
	default:
		caps.ExplainJSON = server.AtLeast(5, 6)
		if server.Probed() && !server.AtLeast(8, 0) {
			caps.Notes = append(caps.Notes, "before MySQL 8.0 there is no atomic DDL: a rehearsal proves shape but not atomicity")
		}
	}
	if !server.Probed() {
		caps.Notes = append(caps.Notes,
			"not probed yet: the version, and therefore EXPLAIN support, is unknown until the target is probed")
	}
	if server.Probed() && declared != "" && declared != server.Flavor {
		caps.Notes = append(caps.Notes, fmt.Sprintf(
			"declared %s but the server reports %s: fix the target so a rehearsal is planned for the right engine",
			declared, server.Flavor))
	}
	return caps
}
