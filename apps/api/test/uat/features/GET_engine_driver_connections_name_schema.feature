Feature: GET /external/v1/engine/{driver}/connections/{name}/schema
  The tables and columns a connection declares.

  Background:
    Given the stack is running
    And a clean registry
    And connection "schema_pg" is registered for driver "postgres"

  Scenario: the schema lists a table a reviewed change created
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_schema_probe (id INT)" and rollback "DROP TABLE uat_schema_probe"
    And I approve the change
    And I apply the change
    Then the schema lists table "uat_schema_probe"

  Scenario: the schema does not invent a table
    Then the schema does not list table "uat_absent_table"

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/connections/schema_pg/schema"
    Then the response status is 401
