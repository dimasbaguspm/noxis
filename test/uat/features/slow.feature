@slow
Feature: Talking to a real engine
  The @fast suite proves refusals and shapes; this one proves the engine work against containers,
  because a fake cannot disagree with you. Run it with UAT_SLOW=1.

  Scenario: the probe measures the container
    Given I hold the ops token
    And I register the "mysql" container as "uat-engine-mysql"
    When I probe the target I registered
    Then the request succeeds
    And the capabilities name the "mysql" engine
    And the capabilities report a version

  Scenario: a read comes back from the container
    Given I hold the ops token
    And I register the "mariadb" container as "uat-engine-mariadb"
    When I run the statement "SELECT 1 AS one"
    Then the request succeeds
    And the result has 1 row

  Scenario: a write is still refused against a reachable engine
    Given I hold the ops token
    And I register the "mysql" container as "uat-engine-refuse"
    When I run the statement "DELETE FROM users"
    Then the request fails with status 400
    And the refusal mentions "delete"

  Scenario: a DDL change is proven in a throwaway database and undone
    Given I hold the ops token
    And I register the "mysql" container as "uat-propose-ddl"
    When I propose "ALTER TABLE users ADD COLUMN starred TINYINT NOT NULL DEFAULT 0" with rollback "ALTER TABLE users DROP COLUMN starred" titled "add starred"
    Then the request succeeds with status 201
    And the proposal is "verified"
    And the rehearsal schema diff mentions "starred"

  Scenario: a rollback that does not restore the schema fails the proposal
    Given I hold the ops token
    And I register the "mariadb" container as "uat-propose-bad-rollback"
    When I propose "ALTER TABLE users ADD COLUMN stars TINYINT NOT NULL DEFAULT 0" with rollback "ALTER TABLE users ADD COLUMN other TINYINT NOT NULL DEFAULT 0" titled "lying rollback"
    Then the request succeeds with status 201
    And the proposal is "failed"
    And the rehearsal note mentions "rollback"

  Scenario: a DML change is rehearsed against sampled rows
    Given I hold the ops token
    And I register the "mysql" container as "uat-propose-dml"
    When I propose "UPDATE users SET note = 'changed' WHERE id = 1" with rollback "UPDATE users SET note = 'one' WHERE id = 1" titled "touch one row" at tier "rehearsal"
    Then the request succeeds with status 201
    And the proposal is "verified"
    And the rehearsal row diff mentions "users"

  Scenario: a destructive change with no rollback is irreversible
    Given I hold the ops token
    And I register the "mariadb" container as "uat-propose-destructive"
    When I propose "DROP TABLE users" titled "drop users"
    Then the request succeeds with status 201
    And the proposal is irreversible

  Scenario: a proposal is approved by one token and applied by another
    Given I hold the ops token
    And I register the "mysql" container as "uat-decide-full"
    When I hold the reviewer token
    And I propose "ALTER TABLE users ADD COLUMN reviewed_by INT NULL" with rollback "ALTER TABLE users DROP COLUMN reviewed_by" titled "add reviewed_by"
    Then the request succeeds with status 201
    And the proposal is "verified"
    And I hold the ops token
    And I approve the proposal
    Then the request succeeds
    And the proposal is "approved"
    And I apply the proposal
    Then the request succeeds
    And the proposal is "applied"
    And the events mention "approved"
    And the events mention "applied"
    And the engine has column "reviewed_by" on table "users"

  Scenario: applying an irreversible change needs an explicit acknowledgement
    Given I hold the ops token
    And I register the "mariadb" container as "uat-decide-destructive"
    When I propose "TRUNCATE TABLE users" titled "empty users"
    Then the request succeeds with status 201
    And the proposal is irreversible
    And I approve the proposal
    Then the request succeeds
    And I apply the proposal
    Then the request fails with status 400
    And I apply the proposal confirming destruction
    Then the request succeeds
    And the proposal is "applied"

  Scenario: a CREATE INDEX is scoped and proven
    Given I hold the ops token
    And I register the "mysql" container as "uat-propose-index"
    When I propose "CREATE INDEX idx_users_note ON users (note)" with rollback "DROP INDEX idx_users_note ON users" titled "index the note column"
    Then the request succeeds with status 201
    And the proposal is "verified"
    And the rehearsal schema diff mentions "idx_users_note"
