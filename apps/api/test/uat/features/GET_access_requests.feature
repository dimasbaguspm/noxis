Feature: GET /external/v1/access-requests
  The queue an admin works from.

  Background:
    Given the stack is running
    And a clean registry
    And connection "queue_pg" is registered for driver "postgres"

  Scenario: a filed request appears in the queue
    When I request the "read" level on the current connection
    Then the access request list contains it

  Scenario: the queue needs a session
    Given no session
    When I send a GET to "/external/v1/access-requests"
    Then the response status is 401
