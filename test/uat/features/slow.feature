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
