Feature: Proposing a change
  A proposal is the statement, the rollback that undoes it, and the evidence that the rollback
  works. The queue only ever holds things that were attempted, so a failed attempt is stored.

  Scenario: a read is not a proposal
    Given I hold the ops token
    And I register a target named "uat-propose-read" on port 1
    When I propose "SELECT 1" titled "nothing"
    Then the request fails with status 400
    And the refusal mentions "changes nothing"

  Scenario: an unreachable target still records the attempt
    # The artifact is the point: a proposal that could not be rehearsed is stored as failed with the
    # reason, and the caller gets it back rather than a transport error.
    Given I hold the ops token
    And I register a target named "uat-propose-down" on port 1
    When I propose "ALTER TABLE users ADD COLUMN note2 VARCHAR(10)" with rollback "ALTER TABLE users DROP COLUMN note2" titled "add note2"
    Then the request succeeds with status 201
    And the proposal is "failed"
    And I list my proposals
    Then the newest proposal is "failed"

  Scenario: an unknown tier is refused
    Given I hold the ops token
    And I register a target named "uat-propose-tier" on port 1
    When I propose "ALTER TABLE users ADD COLUMN note2 VARCHAR(10)" titled "add note2" at tier "wishful"
    Then the request fails with status 400
