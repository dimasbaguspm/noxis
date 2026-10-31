Feature: POST /external/v1/engine/{driver}/connections
  Registering a connection is an upsert on its name; the identity fields are fixed once it exists.

  Background:
    Given the stack is running

  Scenario: a target is registered and listed
    Given connection "upsert_pg" is registered for driver "postgres"
    Then the connection test passes

  Scenario: the identity of an existing connection cannot be rewritten
    Given connection "locked_pg" is registered for driver "postgres"
    Then changing the connection's host is refused

  Scenario: an engine that no module serves is refused
    When I send a POST to "/external/v1/engine/oracle/connections" with body:
      """
      {"name":"nope","driver":"oracle","host":"127.0.0.1","port":1}
      """
    Then the response status is 404

  Scenario: registering needs a session
    Given no session
    When I send a POST to "/external/v1/engine/postgres/connections" with body:
      """
      {"name":"nope","driver":"postgres","host":"127.0.0.1","port":1}
      """
    Then the response status is 401
