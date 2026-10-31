Feature: GET /external/v1/instances
  Every registered connection is addressable by a stable id, across engines.

  Background:
    Given the stack is running

  Scenario: the list contains the registered connection
    Given a clean registry
    And connection "instances_pg" is registered for driver "postgres"
    Then the instances list contains the current connection

  Scenario: the list needs a session
    Given no session
    When I send a GET to "/external/v1/instances"
    Then the response status is 401
