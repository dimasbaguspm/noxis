Feature: POST /external/v1/access-requests
  A member asks for a level on one connection; the ask is recorded and listed.

  Background:
    Given the stack is running
    And a clean registry
    And connection "ask_pg" is registered for driver "postgres"

  Scenario: a request is filed and listed
    When I request the "read" level on the current connection
    Then the access request list contains it

  Scenario: filing needs a session
    Given no session
    When I send a POST to "/external/v1/access-requests" with body:
      """
      {"connection":"ask_pg","level":"read","reason":"uat"}
      """
    Then the response status is 401
