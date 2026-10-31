Feature: DELETE /external/v1/engine/{driver}/connections/{name}/permissions
  Revoking a grant leaves the connection with none.

  Background:
    Given the stack is running
    And a clean registry
    And connection "revoke_pg" is registered for driver "postgres"

  Scenario: a granted level is revoked
    When I grant "local:uat-user@example.com" the "read" level on the current connection
    And I revoke "local:uat-user@example.com" from the current connection
    Then the permission list is empty

  Scenario: revoking needs a session
    Given no session
    When I send a DELETE to "/external/v1/engine/postgres/connections/revoke_pg/permissions?member=local:uat-user@example.com"
    Then the response status is 401
