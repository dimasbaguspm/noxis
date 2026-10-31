Feature: POST /external/v1/engine/{driver}/connections/{name}/operations
  Reads run directly; a write never does, on every SQL engine.

  Background:
    Given the stack is running
    And a clean registry

  Scenario Outline: <driver> answers a read and refuses a write
    Given connection "ops_<driver>" is registered for driver "<driver>"
    When I read "SELECT 6*6 AS n"
    Then the result has 1 row(s)
    And the first row's "n" is "36"
    When a direct write "CREATE TABLE IF NOT EXISTS uat_ops_<driver> (id INT)" is refused
    Then the schema does not list table "uat_ops_<driver>"

    Examples:
      | driver    |
      | postgres  |
      | mysql     |
      | mariadb   |
      | cockroach |
      | sqlite    |

  Scenario Outline: <driver> refuses DDL smuggled behind a read
    Given connection "ops_smuggle_<driver>" is registered for driver "<driver>"
    When I read "SELECT 1; DROP TABLE uat_nothing"
    Then the request is rejected

    Examples:
      | driver    |
      | postgres  |
      | mysql     |
      | mariadb   |
      | cockroach |
      | sqlite    |

  Scenario: an unknown operation is refused
    Given connection "ops_unknown" is registered for driver "postgres"
    When I send an unknown console operation
    Then the request is refused as a bad request

  Scenario: the route needs a session
    Given no session
    When I send a POST to "/external/v1/engine/postgres/connections/anything/operations" with body:
      """
      {"op":"query","sql":"SELECT 1"}
      """
    Then the response status is 401
