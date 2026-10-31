Feature: GET /external/v1/engine/{driver}/connections/{name}/tables/{table}/rows
  A bounded preview of one table's rows.

  Background:
    Given the stack is running
    And a clean registry
    And connection "rows_pg" is registered for driver "postgres"

  Scenario: a table with rows answers with a grid
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_rows_probe (id INT)" and rollback "DROP TABLE uat_rows_probe"
    And I approve the change
    And I apply the change
    When I submit a change with query "INSERT INTO uat_rows_probe (id) VALUES (1), (2)" and rollback "DELETE FROM uat_rows_probe"
    And I approve the change
    And I apply the change
    Then table "uat_rows_probe" has 2 row(s)

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/connections/rows_pg/tables/customers/rows"
    Then the response status is 401
