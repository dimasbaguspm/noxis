package commands

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"noxis/internal/common/release"
	"noxis/internal/constants"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// releaseTag is the tag the publish job moves to every green commit on master, and the image tag that
// stands for the same build.
const releaseTag = "nightly"

// downloadTimeout bounds one asset. The binary is around 30 MB, so this is generous on purpose: a slow link
// is not a reason to leave an operator with half an upgrade.
const downloadTimeout = 5 * time.Minute

// NewUpgrade moves the binary you are running to the published build. It is the CLI's own binary only:
// there is deliberately no route on the API that upgrades a server, because a process cannot know whether
// systemd, compose or a person started it, and an endpoint that rewrites itself is remote execution with a
// changelog.
func NewUpgrade(*Session) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Replace this binary with the published build",
		Args:  cobra.NoArgs,
		// Nothing here talks to the API: it is a file next to this process and a release tag.
		Annotations: map[string]string{constants.AnnoNoToken: "true"},
		Long: "Fetches the published build for this platform, verifies its checksum and replaces the binary " +
			"this process is running. The previous binary stays beside it as `noxis.old`, and the process that " +
			"is running now is the one that keeps running: restart it to use the new build.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return upgrade(cmd.Context(), cmd, check)
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "report both builds and change nothing")
	return cmd
}

// upgrade is the whole command: what is published, what this is, and — unless --check — the swap.
func upgrade(ctx context.Context, cmd *cobra.Command, check bool) error {
	out := cmd.OutOrStdout()
	published, err := publishedInfo(ctx)
	if err != nil {
		return fmt.Errorf("the published build could not be read: %w", err)
	}
	if published == nil {
		return fmt.Errorf("nothing is published at %s; see %s", constants.ReleaseURL, constants.ReleaseURL)
	}
	if _, err := fmt.Fprintf(out, "running   %s (%s/%s)\npublished %s, built %s\n",
		constants.Build(), runtime.GOOS, runtime.GOARCH, publishedString(published), published.BuiltAt); err != nil {
		return err
	}
	if published.Commit == constants.Commit {
		_, err := fmt.Fprintln(out, "this is the published build")
		return err
	}
	if check {
		_, err := fmt.Fprintln(out, "run `noxis upgrade` to replace this binary")
		return err
	}
	return replace(ctx, cmd, assetURL())
}

// replace downloads, verifies and swaps. The verification is not a courtesy: a binary that arrives over the
// network and runs with the operator's scopes is the one download in this product that must be checked.
func replace(ctx context.Context, cmd *cobra.Command, assetURL string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("this binary cannot be located: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("this binary cannot be located: %w", err)
	}
	if runtime.GOOS == "windows" {
		return errors.New("windows is not supported: a running executable cannot be replaced (rename noxis.exe to noxis.exe.old and move the new one in)")
	}
	if inContainer() {
		return fmt.Errorf("this binary came from an image: upgrade the image instead — `docker pull ghcr.io/dimasbaguspm/noxis:%s`", releaseTag)
	}

	dir := filepath.Dir(exe)
	want, err := checksum(ctx, constants.ChecksumsAsset)
	if err != nil {
		return err
	}
	// The new binary lands in the directory it is going to live in, so the swap is a rename on one
	// filesystem: a rename is atomic, a copy across filesystems is not.
	temp, err := os.CreateTemp(dir, ".noxis-upgrade-*")
	if err != nil {
		return fmt.Errorf("%s is not writable: %w", dir, err)
	}
	defer func() { _ = os.Remove(temp.Name()) }()

	written, got, err := download(ctx, assetURL, temp)
	if err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("checksum mismatch: downloaded %s, SHA256SUMS says %s — the file was deleted and nothing was replaced", got, want)
	}
	if err := os.Chmod(temp.Name(), 0o755); err != nil { //nolint:gosec // a binary has to be executable
		return err
	}
	old := exe + ".old"
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("the running binary could not be moved aside: %w", err)
	}
	if err := os.Rename(temp.Name(), exe); err != nil {
		// Put the old one back: a binary that is neither here nor there is an operator with no noxis.
		_ = os.Rename(old, exe)
		return fmt.Errorf("the new binary could not be put in place: %w", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(),
		"downloaded %.1f MiB, checksum ok\nreplaced %s (the previous binary is %s)\nrestart it: this process is still the old build\n",
		float64(written)/(1<<20), exe, old)
	return err
}

// checksum reads the published SHA256SUMS and answers the line for this asset. A file that is not listed
// is a refusal, not a warning.
func checksum(ctx context.Context, listURL string) (string, error) {
	body, err := fetch(ctx, listURL)
	if err != nil {
		return "", fmt.Errorf("the checksums could not be read: %w", err)
	}
	defer func() { _ = body.Close() }()

	name := assetName()
	lines := bufio.NewScanner(body)
	for lines.Scan() {
		fields := strings.Fields(lines.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], nil
		}
	}
	if err := lines.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("SHA256SUMS does not list %s", name)
}

// download writes the asset and answers how many bytes it wrote and what they hash to, in one pass: the
// file is never read twice and never trusted in between.
func download(ctx context.Context, assetURL string, into io.Writer) (int64, string, error) {
	body, err := fetch(ctx, assetURL)
	if err != nil {
		return 0, "", fmt.Errorf("the published binary could not be downloaded: %w", err)
	}
	defer func() { _ = body.Close() }()

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(into, hash), body)
	if err != nil {
		return written, "", err
	}
	return written, hex.EncodeToString(hash.Sum(nil)), nil
}

func fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		cancel()
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		cancel()
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		cancel()
		return nil, fmt.Errorf("%s answered %s", url, response.Status)
	}
	// The caller closes the body; cancelling the timeout with it is what keeps a slow download from
	// outliving the command.
	return &cancelOnClose{ReadCloser: response.Body, cancel: cancel}, nil
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

// publishedInfo reads the artifact the publish job uploads. It is not the API's update notice: that one is
// a courtesy with a three-second budget, and this command is a person waiting for an answer.
func publishedInfo(ctx context.Context) (*release.Info, error) {
	body, err := fetch(ctx, constants.VersionAsset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()

	var info release.Info
	if err := json.NewDecoder(body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// assetName is the file the publish job uploads for this platform: `noxis-linux-amd64`.
func assetName() string { return fmt.Sprintf("noxis-%s-%s", runtime.GOOS, runtime.GOARCH) }

// assetURL is that file's address in the one tag that is published.
func assetURL() string { return constants.ReleaseBase + "/" + assetName() }

// publishedString is the published build the way a person reads this one: `nightly+9f3c2e8`.
func publishedString(info *release.Info) string {
	if info.Commit == "" {
		return info.Version
	}
	return info.Version + "+" + info.Commit
}

// inContainer reports whether this process looks like a container's. It is a guess, and the one place a
// guess is right: the honest answer there is "upgrade the image", and refusing loudly beats rewriting a
// file inside an immutable filesystem.
func inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	raw, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	text := string(raw)
	for _, marker := range []string{"docker", "kubepods", "containerd", "podman", "libpod"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
