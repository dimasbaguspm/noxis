Feature: PUT /external/v1/engine/{driver}/connections/{name}/permissions
  An admin grants a level to a user or a group on one connection.

  Background:
    Given the stack is running
    And a clean registry
    And connection "grant_pg" is registered for driver "postgres"

  Scenario Outline: the <level> level is granted and listed
    When I grant "local:uat-user@example.com" the "<level>" level on the current connection
    Then the permission list shows "local:uat-user@example.com" at "<level>"

    Examples:
      | level |
      | read  |
      | write |
      | admin |

  Scenario: granting needs a session
    Given no session
    When I send a PUT to "/external/v1/engine/postgres/connections/grant_pg/permissions" with body:
      """
      {"member":"local:uat-user@example.com","memberType":"user","level":"read"}
      """
    Then the response status is 401
