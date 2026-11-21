package constants

// Version and Commit are the two halves of what this binary is, and both are stamped at link time:
//
//	go build -ldflags "-X noxis/internal/constants.Version=nightly \
//	                   -X noxis/internal/constants.Commit=$(git rev-parse --short=7 HEAD)"
//
// Version is the release line — `nightly` for a build of master, a semver tag for a release — and Commit
// is the seven characters a person quotes in a bug report. A build nobody stamped says `dev` and says it
// has no commit, instead of pretending to be a release.
var (
	Version = "dev"
	Commit  = "unknown"
)

// ReleaseURL is where every update notice points: the releases page, never a raw asset.
const ReleaseURL = "https://github.com/dimasbaguspm/noxis/releases"

// ReleaseBase is where the publish job puts every artifact of that one tag: the two binaries, the
// checksums beside them, and version.json.
const ReleaseBase = "https://github.com/dimasbaguspm/noxis/releases/download/nightly"

// VersionAsset is the artifact an update check reads: a signed-out fetch of a release asset costs no API
// rate limit and cannot be scraped wrong.
const VersionAsset = ReleaseBase + "/version.json"

// ChecksumsAsset lists the SHA-256 of both binaries. An upgrade that does not check it is a download that
// trusts whatever answered.
const ChecksumsAsset = ReleaseBase + "/SHA256SUMS"

// Build is the one string a person reads: `nightly+db92d5a`, `v2.0.0+db92d5a`, or `dev`.
func Build() string {
	if Commit == "" || Commit == "unknown" {
		return Version
	}
	return Version + "+" + Commit
}
