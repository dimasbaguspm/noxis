Feature: GET /external/v1/instances/counts
  The console's sidebar counts what is registered, per driver.

  Background:
    Given the stack is running

  Scenario: the counts answer for a registered connection
    Given a clean registry
    And connection "counts_pg" is registered for driver "postgres"
    When I send a GET to "/external/v1/instances/counts"
    Then the response status is 200
    And the response body contains "postgres"

  Scenario: the counts need a session
    Given no session
    When I send a GET to "/external/v1/instances/counts"
    Then the response status is 401
