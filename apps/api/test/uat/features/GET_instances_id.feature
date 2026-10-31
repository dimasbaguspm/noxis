Feature: GET /external/v1/instances/{id}
  One instance by its stable id.

  Background:
    Given the stack is running

  Scenario: the registered instance opens
    Given a clean registry
    And connection "instance_pg" is registered for driver "postgres"
    And the instances list contains the current connection
    Then I open the current instance

  Scenario: an unknown id is refused
    Then opening an unknown instance is refused

  Scenario: the detail needs a session
    Given no session
    When I send a GET to "/external/v1/instances/00000000-0000-0000-0000-000000000000"
    Then the response status is 401
