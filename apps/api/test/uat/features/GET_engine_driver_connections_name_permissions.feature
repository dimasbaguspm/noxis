Feature: GET /external/v1/engine/{driver}/connections/{name}/permissions
  The grants on one connection, which the console renders as the access panel.

  Background:
    Given the stack is running
    And a clean registry
    And connection "grants_pg" is registered for driver "postgres"

  Scenario: the list shows a grant and then nothing after a revoke
    When I grant "local:uat-user@example.com" the "read" level on the current connection
    Then the permission list shows "local:uat-user@example.com" at "read"
    When I revoke "local:uat-user@example.com" from the current connection
    Then the permission list is empty

  Scenario: the list needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/connections/grants_pg/permissions"
    Then the response status is 401
