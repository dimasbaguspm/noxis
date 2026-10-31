Feature: GET /external/v1/engine/{driver}/connections/{name}
  One connection's detail, with its password redacted.

  Background:
    Given the stack is running

  Scenario: the detail hides the password
    Given connection "detail_pg" is registered for driver "postgres"
    Then the current connection still hides its password

  Scenario: an unknown name is refused
    Given connection "detail_pg" is registered for driver "postgres"
    When I send a GET to "/external/v1/engine/postgres/connections/absent_connection"
    Then the response status is 404

  Scenario: the detail needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/connections/anything"
    Then the response status is 401
