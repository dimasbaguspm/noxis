Feature: Probing a target
  A stored target is a set of strings until something connects to it. The probe is the only
  route that opens a connection on demand, and it records what it found — or that it failed —
  on the target itself. Capabilities answer from what is already known, without connecting.

  Scenario: capabilities answer without a connection
    Given I hold the ops token
    And I register a target named "uat-caps"
    When I hold the agent token
    And I ask for the capabilities of the target I registered
    Then the request succeeds
    And the capabilities name the "mysql" engine
    And the capabilities say it has not been probed

  Scenario: a probe that cannot connect records the failure
    Given I hold the ops token
    When I register a target named "uat-unreachable" on port 1
    And I probe the target I registered
    Then the request fails with status 422
    And the target reports probe status "failed"

  Scenario: an archived target is not probed
    Given I hold the ops token
    When I register a target named "uat-archived-probe"
    And I archive the target I registered
    And I probe the target I registered
    Then the request fails with status 409
