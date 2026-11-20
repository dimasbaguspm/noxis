Feature: Reading through a target
  The read route is read-only by construction: a statement is classified before a connection is
  opened, so a write is refused even when the target is reachable, and nothing is executed.

  Scenario: a write is refused before a connection is attempted
    Given I hold the ops token
    And I register a target named "uat-read" on port 1
    When I run the statement "DELETE FROM users"
    Then the request fails with status 400
    And the refusal mentions "delete"

  Scenario: an archived target cannot be read
    Given I hold the ops token
    And I register a target named "uat-read-archived" on port 1
    When I archive the target I registered
    Then the request succeeds
    And I run the statement "SELECT 1"
    Then the request fails with status 409
    And the refusal mentions "archived"

  Scenario: a second statement in one request is refused
    Given I hold the ops token
    And I register a target named "uat-read-two" on port 1
    When I run the statement "SELECT 1; DROP TABLE users"
    Then the request fails with status 400
    And the refusal mentions "one statement per request"

  Scenario: a locking read is refused
    Given I hold the ops token
    And I register a target named "uat-read-lock" on port 1
    When I run the statement "SELECT * FROM users FOR UPDATE"
    Then the request fails with status 400
    And the refusal mentions "locking read"
