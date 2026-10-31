Feature: GET /external/v1/access
  An admin sees the whole access landscape without holding a grant on each connection.

  Background:
    Given the stack is running
    And a clean registry
    And connection "access_pg" is registered for driver "postgres"

  Scenario: the matrix lists the connection
    Then the access matrix lists the current connection

  Scenario: the matrix needs a session
    Given no session
    When I send a GET to "/external/v1/access"
    Then the response status is 401
