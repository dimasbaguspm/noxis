Feature: GET /healthz
  The compose stack waits on this route, so it answers before anyone signs in.

  Background:
    Given the stack is running

  Scenario: the process reports itself healthy
    Then the health endpoint reports ok

  Scenario: an unknown API path answers JSON instead of the console
    When I request "/external/v1/nope"
    Then the response status is 404
    And the response header "Content-Type" is "application/json"
