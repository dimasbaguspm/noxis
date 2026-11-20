Feature: Targets
  A target is one MySQL or MariaDB server this noxis governs. Registering one needs a token
  that may administer; reading needs only `read`. The password goes in and never comes back
  out, and a target is archived rather than deleted.

  Scenario: an operator registers a target and gets its reference back
    Given I hold the ops token
    When I register a target named "uat-primary"
    Then the request succeeds with status 201
    And the target reference is a reference
    And the password is stored
    And the response carries no password

  Scenario: an agent may not register a target
    Given I hold the agent token
    When I register a target named "uat-agent-made"
    Then the request fails with status 403

  Scenario: the list pages by cursor, newest first
    Given I hold the ops token
    And I register three targets
    When I list targets two at a time
    Then I see 2 targets
    And there is a next cursor
    When I follow the cursor
    Then I see 1 target
    And there is no next cursor

  Scenario: an unknown reference is a 404 and a non-reference is a 400
    Given I hold the ops token
    When I read the target "0197a1b2-c3d4-7e8f-9a0b-1c2d3e4f5a6b"
    Then the request fails with status 404
    When I read the target "42"
    Then the request fails with status 400

  Scenario: archiving twice is a conflict, and the archived target leaves the list
    Given I hold the ops token
    When I register a target named "uat-archived"
    And I archive the target I registered
    Then the request succeeds
    And I archive the target I registered
    Then the request fails with status 409
