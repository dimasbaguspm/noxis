// Package steps wires the Gherkin sentences to the API client. Every step talks HTTP to the
// running binary: nothing here opens the state file or a database.
package steps

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"noxis/internal/client"
	"noxis/internal/common/ref"
	"noxis/internal/constants"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

// world is one scenario's state.
type world struct {
	api       *client.Client
	identity  client.Identity
	health    client.Health
	target    client.Target
	targets   []client.Target
	rows      client.Rows
	proposal  client.Proposal
	proposals []client.Proposal
	caps      client.Capabilities
	cursor    string
	// The retry scenario keeps what the first call made, so the second answer has something to equal.
	firstRef    string
	retryStatus int
	err         error
}

// InitializeScenario registers every step against a fresh world.
func InitializeScenario(ctx *godog.ScenarioContext) {
	w := &world{}
	ctx.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
		w.identity, w.health = client.Identity{}, client.Health{}
		w.target, w.targets, w.cursor, w.err = client.Target{}, nil, "", nil
		w.rows = client.Rows{}
		w.proposal, w.proposals = client.Proposal{}, nil
		w.caps = client.Capabilities{}
		w.api, w.err = client.New(os.Getenv("UAT_URL"), "")
		return c, w.err
	})

	// identity
	ctx.Step(`^I hold the (agent|reviewer|ops) token$`, w.holdToken)
	ctx.Step(`^I ask who I am$`, w.askWhoami)
	ctx.Step(`^I ask who I am without a token$`, w.askWhoamiWithoutToken)
	ctx.Step(`^I ask for the health of the server without a token$`, w.askHealth)
	ctx.Step(`^my name is "([^"]*)"$`, w.assertName)
	ctx.Step(`^my scopes are "([^"]*)"$`, w.assertScopes)
	ctx.Step(`^the reported version names the build$`, w.assertVersionNamesTheBuild)

	// targets
	ctx.Step(`^I register a target named "([^"]*)"$`, w.registerTarget)
	ctx.Step(`^I register three targets$`, w.registerThree)
	ctx.Step(`^I register a target named "([^"]*)" twice under the key "([^"]*)"$`, w.registerTwice)
	ctx.Step(`^the retry answers the target the first call made$`, w.assertRetryAnsweredTheFirst)
	ctx.Step(`^the retry answers with status (\d+)$`, w.assertRetryStatus)
	ctx.Step(`^the target reference is a reference$`, w.assertTargetRef)
	ctx.Step(`^the password is stored$`, w.assertPasswordStored)
	ctx.Step(`^the response carries no password$`, w.assertNoPasswordOnWire)
	ctx.Step(`^I list targets two at a time$`, w.listTwo)
	ctx.Step(`^I follow the cursor$`, w.followCursor)
	ctx.Step(`^I see (\d+) targets?$`, w.assertCount)
	ctx.Step(`^there is a next cursor$`, w.assertHasCursor)
	ctx.Step(`^there is no next cursor$`, w.assertNoCursor)
	ctx.Step(`^I read the target "([^"]*)"$`, w.readTarget)
	ctx.Step(`^I archive the target I registered$`, w.archiveTarget)

	// probe
	ctx.Step(`^I register a target named "([^"]*)" on port (\d+)$`, w.registerTargetOnPort)
	ctx.Step(`^I ask for the capabilities of the target I registered$`, w.askCapabilities)
	ctx.Step(`^I probe the target I registered$`, w.probeTarget)
	ctx.Step(`^the capabilities name the "([^"]*)" engine$`, w.assertFlavor)
	ctx.Step(`^the capabilities say it has not been probed$`, w.assertNotProbed)
	ctx.Step(`^the target reports probe status "([^"]*)"$`, w.assertProbeStatus)

	// reads
	ctx.Step(`^I run the statement "([^"]*)"$`, w.runStatement)
	ctx.Step(`^the refusal mentions "([^"]*)"$`, w.assertRefusalMentions)
	ctx.Step(`^the result has (\d+) rows?$`, w.assertResultRows)

	// proposals
	ctx.Step(`^I propose "([^"]*)" titled "([^"]*)"$`, w.proposeWithoutRollback)
	ctx.Step(`^I propose "([^"]*)" with rollback "([^"]*)" titled "([^"]*)"$`, w.proposeWithRollback)
	ctx.Step(`^I propose "([^"]*)" titled "([^"]*)" at tier "([^"]*)"$`, w.proposeAtTier)
	ctx.Step(`^I propose "([^"]*)" with rollback "([^"]*)" titled "([^"]*)" at tier "([^"]*)"$`, w.proposeWithRollbackAtTier)
	ctx.Step(`^the proposal is "([^"]*)"$`, w.assertProposalStatus)
	ctx.Step(`^the proposal is irreversible$`, w.assertIrreversible)
	ctx.Step(`^the rehearsal schema diff mentions "([^"]*)"$`, w.assertSchemaDiffMentions)
	ctx.Step(`^the rehearsal row diff mentions "([^"]*)"$`, w.assertRowDiffMentions)
	ctx.Step(`^the rehearsal note mentions "([^"]*)"$`, w.assertRehearsalNoteMentions)
	ctx.Step(`^I list my proposals$`, w.listMyProposals)
	ctx.Step(`^the newest proposal is "([^"]*)"$`, w.assertNewestProposal)
	ctx.Step(`^I approve the proposal$`, w.approveProposal)
	ctx.Step(`^I reject the proposal$`, w.rejectProposal)
	ctx.Step(`^I apply the proposal$`, func() error { return w.applyProposal(false) })
	ctx.Step(`^I apply the proposal confirming destruction$`, w.applyConfirming)
	ctx.Step(`^the events mention "([^"]*)"$`, w.assertEventsMention)
	ctx.Step(`^the engine has column "([^"]*)" on table "([^"]*)"$`, w.assertEngineColumn)
	ctx.Step(`^I poll the proposal until it is "([^"]*)"$`, w.pollProposal)
	ctx.Step(`^the receipt names the algorithm "([^"]*)"$`, w.assertAlgorithm)

	// slow
	ctx.Step(`^I register the "(mysql|mariadb)" container as "([^"]*)"$`, w.registerContainer)
	ctx.Step(`^the capabilities report a version$`, w.assertVersionKnown)

	// shared assertions
	ctx.Step(`^the request succeeds$`, w.assertSuccess)
	ctx.Step(`^the request succeeds with status (\d+)$`, w.assertSuccessStatus)
	ctx.Step(`^the request fails with status (\d+)$`, w.assertStatus)
}

func (w *world) holdToken(name string) error {
	secret := map[string]string{
		"agent":    os.Getenv("UAT_AGENT_SECRET"),
		"reviewer": os.Getenv("UAT_REV_SECRET"),
		"ops":      os.Getenv("UAT_OPS_SECRET"),
	}[name]
	if secret == "" {
		return fmt.Errorf("no %s secret in the environment", name)
	}
	api, err := client.New(os.Getenv("UAT_URL"), secret)
	w.api = api
	return err
}

func (w *world) askWhoami() error {
	w.identity, w.err = w.api.Whoami(context.Background())
	return nil
}

func (w *world) askWhoamiWithoutToken() error {
	anonymous, err := client.New(os.Getenv("UAT_URL"), "")
	if err != nil {
		return err
	}
	w.identity, w.err = anonymous.Whoami(context.Background())
	return nil
}

func (w *world) askHealth() error {
	w.health, w.err = w.api.Live(context.Background())
	return nil
}

func (w *world) assertName(want string) error {
	if w.err != nil {
		return w.err
	}
	if w.identity.Name != want {
		return fmt.Errorf("name = %q, want %q", w.identity.Name, want)
	}
	return nil
}

func (w *world) assertScopes(want string) error {
	if w.err != nil {
		return w.err
	}
	if got := strings.Join(w.identity.Scopes, ", "); got != want {
		return fmt.Errorf("scopes = %q, want %q", got, want)
	}
	return nil
}

// assertVersionNamesTheBuild checks the shape, not a literal: the version is stamped at link time, so a
// test that pins it to one string fails on every build that is not that build.
func (w *world) assertVersionNamesTheBuild() error {
	if w.err != nil {
		return w.err
	}
	build := regexp.MustCompile(`^(dev|nightly|v\d+\.\d+\.\d+)(\+[0-9a-f]{7,40})?$`)
	if !build.MatchString(w.health.Version) {
		return fmt.Errorf("version = %q, want a release line and a commit", w.health.Version)
	}
	return nil
}

// registerTarget registers one target with a password, so the sealing path is exercised.
func (w *world) registerTarget(name string) error {
	w.target, w.err = w.api.CreateTarget(context.Background(), client.CreateTargetInput{
		Name:     name,
		Driver:   constants.DriverMySQL,
		Host:     "127.0.0.1",
		Port:     constants.DefaultMySQLPort,
		Database: "uat",
		Username: "uat",
		Password: "uat-password",
		TLSMode:  constants.TLSPreferred,
	})
	return nil
}

func (w *world) registerThree() error {
	for _, name := range []string{"uat-paged-a", "uat-paged-b", "uat-paged-c"} {
		if err := w.registerTarget(name); err != nil {
			return err
		}
		if w.err != nil {
			return w.err
		}
	}
	return nil
}

func (w *world) assertTargetRef() error {
	if w.err != nil {
		return w.err
	}
	if !ref.Valid(w.target.Ref) {
		return fmt.Errorf("ref = %q, not a reference", w.target.Ref)
	}
	return nil
}

func (w *world) assertPasswordStored() error {
	if w.err != nil {
		return w.err
	}
	if !w.target.HasPassword {
		return errors.New("the target reports no password, so it was not sealed")
	}
	return nil
}

// assertNoPasswordOnWire reads the target again with a bare request and checks the body,
// because a typed client cannot prove what the server actually sent.
func (w *world) assertNoPasswordOnWire() error {
	if w.err != nil {
		return w.err
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		fmt.Sprintf("%s/v1/targets/%s", os.Getenv("UAT_URL"), w.target.Ref), http.NoBody)
	if err != nil {
		return err
	}
	request.Header.Set(constants.HeaderAuthorization, constants.SchemeBearer+" "+os.Getenv("UAT_OPS_SECRET"))
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	for _, leaked := range []string{"password_cipher", `"password":`, "uat-password", "v1:"} {
		if strings.Contains(string(body), leaked) {
			return fmt.Errorf("the response carries %q", leaked)
		}
	}
	return nil
}

// listTwo pages only its own three targets, because scenarios share one server and the
// state file grows across them.
func (w *world) listTwo() error {
	page, err := w.api.ListTargets(context.Background(), client.ListTargetsInput{Limit: 2, Query: "uat-paged"})
	w.targets, w.cursor, w.err = page.Items, page.NextCursor, err
	return nil
}

func (w *world) followCursor() error {
	if w.err != nil {
		return w.err
	}
	page, err := w.api.ListTargets(context.Background(), client.ListTargetsInput{Limit: 2, Query: "uat-paged", Cursor: w.cursor})
	w.targets, w.cursor, w.err = page.Items, page.NextCursor, err
	return nil
}

func (w *world) assertCount(want int) error {
	if w.err != nil {
		return w.err
	}
	if len(w.targets) != want {
		return fmt.Errorf("saw %d targets, want %d", len(w.targets), want)
	}
	return nil
}

func (w *world) assertHasCursor() error {
	if w.err != nil {
		return w.err
	}
	if w.cursor == "" {
		return errors.New("no next cursor on a full page")
	}
	return nil
}

func (w *world) assertNoCursor() error {
	if w.err != nil {
		return w.err
	}
	if w.cursor != "" {
		return fmt.Errorf("next cursor %q on the last page", w.cursor)
	}
	return nil
}

func (w *world) readTarget(value string) error {
	_, w.err = w.api.GetTarget(context.Background(), value)
	return nil
}

func (w *world) archiveTarget() error {
	if w.err != nil {
		return w.err
	}
	_, w.err = w.api.ArchiveTarget(context.Background(), w.target.Ref)
	return nil
}

func (w *world) assertSuccess() error { return w.err }

func (w *world) assertSuccessStatus(want int) error {
	if w.err != nil {
		return w.err
	}
	if got := w.api.LastStatus(); got != want {
		return fmt.Errorf("status = %d, want %d", got, want)
	}
	return nil
}

func (w *world) assertStatus(want int) error {
	if w.err == nil {
		return fmt.Errorf("the request succeeded, want status %d", want)
	}
	var apiErr *client.Error
	if !errors.As(w.err, &apiErr) {
		return w.err
	}
	if apiErr.Status != want {
		return fmt.Errorf("status = %d, want %d", apiErr.Status, want)
	}
	return nil
}

// registerTargetOnPort registers a target on a chosen port, so a probe can be pointed at a
// socket nothing is listening on and fail in a way the test controls.
func (w *world) registerTargetOnPort(name string, port int) error {
	w.target, w.err = w.api.CreateTarget(context.Background(), client.CreateTargetInput{
		Name:     name,
		Driver:   constants.DriverMySQL,
		Host:     "127.0.0.1",
		Port:     port,
		Database: "uat",
		Username: "uat",
		Password: "uat-password",
		TLSMode:  constants.TLSPreferred,
	})
	return nil
}

func (w *world) askCapabilities() error {
	w.caps, w.err = w.api.TargetCapabilities(context.Background(), w.target.Ref)
	return nil
}

func (w *world) probeTarget() error {
	result, err := w.api.ProbeTarget(context.Background(), w.target.Ref)
	w.caps, w.err = result.Capabilities, err
	return nil
}

func (w *world) assertFlavor(want string) error {
	if w.err != nil {
		return w.err
	}
	if w.caps.Flavor != want {
		return fmt.Errorf("flavor = %q, want %q (notes: %v)", w.caps.Flavor, want, w.caps.Notes)
	}
	return nil
}

func (w *world) assertNotProbed() error {
	if w.err != nil {
		return w.err
	}
	if w.caps.Version != "" {
		return fmt.Errorf("capabilities report version %q for an unprobed target", w.caps.Version)
	}
	for _, note := range w.caps.Notes {
		if strings.Contains(note, "not probed yet") {
			return nil
		}
	}
	return fmt.Errorf("capabilities do not say the target has not been probed: %v", w.caps.Notes)
}

func (w *world) assertProbeStatus(want string) error {
	stored, err := w.api.GetTarget(context.Background(), w.target.Ref)
	if err != nil {
		return err
	}
	if stored.LastProbeStatus != want {
		return fmt.Errorf("probe status = %q, want %q", stored.LastProbeStatus, want)
	}
	if stored.LastProbeAt == nil {
		return errors.New("a probe that ran left no timestamp")
	}
	return nil
}

// runStatement sends one statement to a target and remembers how it went.
func (w *world) runStatement(statement string) error {
	w.rows, w.err = w.api.Query(context.Background(), w.target.Ref, statement, 0)
	return nil
}

// assertRefusalMentions checks the refusal text, which is where the server explains itself.
func (w *world) assertRefusalMentions(want string) error {
	if w.err == nil {
		return fmt.Errorf("the statement was accepted, want a refusal mentioning %q", want)
	}
	var apiErr *client.Error
	if errors.As(w.err, &apiErr) {
		if strings.Contains(apiErr.Message, want) {
			return nil
		}
		return fmt.Errorf("refusal %q does not mention %q", apiErr.Message, want)
	}
	return w.err
}

// registerContainer registers one of the engines the @slow compose file boots.
func (w *world) registerContainer(engine, name string) error {
	// The harness boots the engine on a port the runtime picks and hands it over here, so two slow
	// runs can happen at once.
	host := os.Getenv("UAT_" + strings.ToUpper(engine) + "_HOST")
	port, err := strconv.Atoi(os.Getenv("UAT_" + strings.ToUpper(engine) + "_PORT"))
	if err != nil || host == "" {
		return fmt.Errorf("the %s engine was not handed over by the harness", engine)
	}
	w.target, w.err = w.api.CreateTarget(context.Background(), client.CreateTargetInput{
		Name:     name,
		Driver:   engine,
		Host:     host,
		Port:     port,
		Database: os.Getenv("UAT_DB_NAME"),
		Username: os.Getenv("UAT_DB_USER"),
		Password: os.Getenv("UAT_DB_PASSWORD"),
		TLSMode:  constants.TLSPreferred,
	})
	return nil
}

func (w *world) assertVersionKnown() error {
	if w.err != nil {
		return w.err
	}
	if w.caps.Version == "" {
		return errors.New("the capabilities report no version after a probe")
	}
	return nil
}

func (w *world) assertResultRows(want int) error {
	if w.err != nil {
		return w.err
	}
	if w.rows.RowCount != want {
		return fmt.Errorf("result has %d row(s), want %d", w.rows.RowCount, want)
	}
	return nil
}

// propose sends one proposal and remembers the verdict it came back with.
func (w *world) propose(title, query, rollback, tier string) error {
	w.proposal, w.err = w.api.CreateProposal(context.Background(), w.target.Ref, client.CreateProposalInput{
		Title: title, Query: query, Rollback: rollback, Tier: tier,
	})
	return nil
}

func (w *world) proposeWithRollback(query, rollback, title string) error {
	return w.propose(title, query, rollback, "")
}

func (w *world) proposeWithRollbackAtTier(query, rollback, title, tier string) error {
	return w.propose(title, query, rollback, tier)
}

func (w *world) proposeAtTier(query, title, tier string) error {
	return w.propose(title, query, "", tier)
}

func (w *world) proposeWithoutRollback(query, title string) error {
	return w.propose(title, query, "", "")
}

// pollProposal reads the proposal until it reaches the status. A 202 is a promise that the answer is
// somewhere else, and this is the only way to keep it.
func (w *world) pollProposal(want string) error {
	if w.err != nil {
		return w.err
	}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		proposal, err := w.api.GetProposal(context.Background(), w.proposal.Ref)
		if err != nil {
			return err
		}
		w.proposal = proposal
		if proposal.Status == want {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("proposal stayed %q for 90s, want %q", w.proposal.Status, want)
}

// assertAlgorithm checks the receipt names what the engine did, not what was expected of it.
func (w *world) assertAlgorithm(want string) error {
	if w.proposal.Algorithm != want {
		return fmt.Errorf("receipt says algorithm %q, want %q", w.proposal.Algorithm, want)
	}
	return nil
}

func (w *world) assertProposalStatus(want string) error {
	if w.err != nil {
		return w.err
	}
	if w.proposal.Status != want {
		return fmt.Errorf("proposal status = %q, want %q (failure: %s)", w.proposal.Status, want, w.proposal.Failure)
	}
	return nil
}

func (w *world) assertIrreversible() error {
	if w.err != nil {
		return w.err
	}
	if !w.proposal.Irreversible {
		return errors.New("the proposal does not report itself irreversible")
	}
	return nil
}

func (w *world) assertSchemaDiffMentions(want string) error {
	if w.proposal.Rehearsal == nil {
		return errors.New("the proposal came back with no rehearsal")
	}
	if !strings.Contains(w.proposal.Rehearsal.SchemaDiff, want) {
		return fmt.Errorf("schema diff does not mention %q:\n%s", want, w.proposal.Rehearsal.SchemaDiff)
	}
	return nil
}

func (w *world) assertRowDiffMentions(want string) error {
	if w.proposal.Rehearsal == nil {
		return errors.New("the proposal came back with no rehearsal")
	}
	if !strings.Contains(w.proposal.Rehearsal.RowDiff, want) {
		return fmt.Errorf("row diff does not mention %q:\n%s", want, w.proposal.Rehearsal.RowDiff)
	}
	return nil
}

func (w *world) assertRehearsalNoteMentions(want string) error {
	if w.proposal.Rehearsal == nil {
		return errors.New("the proposal came back with no rehearsal")
	}
	if !strings.Contains(w.proposal.Rehearsal.Note, want) {
		return fmt.Errorf("rehearsal note does not mention %q: %s", want, w.proposal.Rehearsal.Note)
	}
	return nil
}

func (w *world) listMyProposals() error {
	page, err := w.api.ListProposals(context.Background(), 0, "", "", "", "", true)
	w.proposals, w.err = page.Items, err
	return nil
}

func (w *world) assertNewestProposal(want string) error {
	if w.err != nil {
		return w.err
	}
	if len(w.proposals) == 0 {
		return errors.New("the queue is empty")
	}
	if newest := w.proposals[0].Status; newest != want {
		return fmt.Errorf("newest proposal is %q, want %q", newest, want)
	}
	return nil
}

// decide records one decision on the proposal the world holds.
// A refused decision must not overwrite the proposal the world holds: the next step needs the ref,
// and a zero value would send it to a different URL. Only a success replaces the artifact.
func (w *world) approveProposal() error {
	decided, err := w.api.ApproveProposal(context.Background(), w.proposal.Ref, "looks right")
	w.err = err
	if err == nil {
		w.proposal = decided
	}
	return nil
}

func (w *world) rejectProposal() error {
	decided, err := w.api.RejectProposal(context.Background(), w.proposal.Ref, "not now")
	w.err = err
	if err == nil {
		w.proposal = decided
	}
	return nil
}

func (w *world) applyProposal(confirm bool) error {
	applied, err := w.api.ApplyProposal(context.Background(), w.proposal.Ref, confirm)
	w.err = err
	if err == nil {
		w.proposal = applied
	}
	return nil
}

func (w *world) applyConfirming() error { return w.applyProposal(true) }

func (w *world) assertEventsMention(kind string) error {
	events, err := w.api.ProposalEvents(context.Background(), w.proposal.Ref)
	if err != nil {
		return err
	}
	for _, event := range events {
		if event.Kind == kind {
			return nil
		}
	}
	return fmt.Errorf("no %q event among %d", kind, len(events))
}

// assertEngineColumn asks the engine directly, through the read route, whether the change landed.
func (w *world) assertEngineColumn(column, table string) error {
	rows, err := w.api.Query(context.Background(), w.target.Ref,
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '"+table+
			"' AND column_name = '"+column+"'", 0)
	if err != nil {
		return err
	}
	if rows.RowCount != 1 || len(rows.Values[0]) != 1 || rows.Values[0][0] == nil {
		return fmt.Errorf("unexpected answer: %+v", rows)
	}
	if *rows.Values[0][0] != "1" {
		return fmt.Errorf("the engine reports %s.%s does not exist", table, column)
	}
	return nil
}

// registerTwice sends one registration and then the same key under a different name: if the server had
// made a second row the refs would differ, so equality is the proof that a retry is one row.
func (w *world) registerTwice(name, key string) error {
	first, err := w.api.CreateTarget(context.Background(), client.CreateTargetInput{
		Name:           name,
		Driver:         constants.DriverMySQL,
		Host:           "127.0.0.1",
		Port:           constants.DefaultMySQLPort,
		Database:       "uat",
		Username:       "uat",
		Password:       "uat-password",
		TLSMode:        constants.TLSPreferred,
		IdempotencyKey: key,
	})
	if err != nil {
		w.err = err
		return nil //nolint:nilerr // the harness keeps the failure in w.err for the step to read
	}
	w.firstRef = first.Ref
	w.target, w.err = w.api.CreateTarget(context.Background(), client.CreateTargetInput{
		Name:           name + "-again",
		Driver:         constants.DriverMySQL,
		Host:           "127.0.0.1",
		Port:           constants.DefaultMySQLPort,
		Database:       "uat",
		Username:       "uat",
		Password:       "uat-password",
		TLSMode:        constants.TLSPreferred,
		IdempotencyKey: key,
	})
	w.retryStatus = w.api.LastStatus()
	return nil
}

func (w *world) assertRetryAnsweredTheFirst() error {
	if w.err != nil {
		return w.err
	}
	if w.target.Ref != w.firstRef {
		return fmt.Errorf("the retry answered %q, want the first target %q", w.target.Ref, w.firstRef)
	}
	return nil
}

func (w *world) assertRetryStatus(want string) error {
	status, err := strconv.Atoi(want)
	if err != nil {
		return err
	}
	if w.err != nil {
		return w.err
	}
	if w.retryStatus != status {
		return fmt.Errorf("the retry answered %d, want %d", w.retryStatus, status)
	}
	return nil
}
