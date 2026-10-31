Feature: DELETE /external/v1/engine/{driver}/connections/{name}
  Deleting a connection removes it, and a second delete finds nothing.

  Background:
    Given the stack is running

  Scenario: a connection is deleted and no longer answers
    Given connection "delete_pg" is registered for driver "postgres"
    When I delete the current connection
    Then the connection is gone

  Scenario: deleting it twice is refused
    Given connection "delete_twice_pg" is registered for driver "postgres"
    When I delete the current connection
    And I send a DELETE to "/external/v1/engine/postgres/connections/delete_twice_pg"
    Then the response status is 404

  Scenario: deleting needs a session
    Given no session
    When I send a DELETE to "/external/v1/engine/postgres/connections/anything"
    Then the response status is 401
