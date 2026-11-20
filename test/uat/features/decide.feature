Feature: Deciding a proposal
  An approval is a human's name on evidence. The token that proposed a change cannot be the token
  that approves it, unless it is an admin — and that exception is a journal line of its own.

  Scenario: the proposer cannot approve its own change
    Given I hold the ops token
    And I register a target named "uat-decide-self" on port 1
    When I hold the reviewer token
    And I propose "ALTER TABLE users ADD COLUMN a INT" with rollback "ALTER TABLE users DROP COLUMN a" titled "add a" at tier "static"
    Then the request succeeds with status 201
    And the proposal is "verified"
    And I approve the proposal
    Then the request fails with status 403

  Scenario: a token without the approve scope is refused
    Given I hold the ops token
    And I register a target named "uat-decide-scope" on port 1
    When I hold the reviewer token
    And I propose "ALTER TABLE users ADD COLUMN b INT" with rollback "ALTER TABLE users DROP COLUMN b" titled "add b" at tier "static"
    Then the request succeeds with status 201
    And I hold the agent token
    And I approve the proposal
    Then the request fails with status 403

  Scenario: a proposal whose rehearsal did not prove cannot be approved
    Given I hold the ops token
    And I register a target named "uat-decide-failed" on port 1
    When I hold the reviewer token
    And I propose "ALTER TABLE users ADD COLUMN c INT" with rollback "ALTER TABLE users DROP COLUMN c" titled "add c"
    Then the request succeeds with status 201
    And the proposal is "failed"
    And I approve the proposal
    Then the request fails with status 409

  Scenario: applying something that is not approved is refused
    Given I hold the ops token
    And I register a target named "uat-decide-unapproved" on port 1
    When I propose "ALTER TABLE users ADD COLUMN d INT" with rollback "ALTER TABLE users DROP COLUMN d" titled "add d"
    Then the request succeeds with status 201
    And I apply the proposal
    Then the request fails with status 409
