Feature: POST /external/v1/engine/{driver}/connections/{name}/operations
  Reads run directly; a write never does, and an unknown operation is refused by name.

  Background:
    Given the stack is running
    And a clean registry
    And connection "ops_pg" is registered for driver "postgres"

  Scenario: a select returns its grid
    When I read "SELECT 6*6 AS n"
    Then the result has 1 row(s)
    And the first row's "n" is "36"

  Scenario: a write is refused by the read-only guard
    When a direct write "CREATE TABLE IF NOT EXISTS uat_direct (id INT)" is refused
    Then the schema does not list table "uat_direct"

  Scenario: DML smuggled behind a read is refused
    When I read "SELECT 1; DROP TABLE items"
    Then the request is rejected

  Scenario: an unknown operation is refused
    When I send an unknown console operation
    Then the request is refused as a bad request

  Scenario: the route needs a session
    Given no session
    When I send a POST to "/external/v1/engine/postgres/connections/ops_pg/operations" with body:
      """
      {"op":"query","sql":"SELECT 1"}
      """
    Then the response status is 401
