Feature: GET /external/v1/engine/{driver}/connections/{name}/schema
  The tables and columns a connection declares, on every engine that declares them.

  Background:
    Given the stack is running
    And a clean registry

  Scenario Outline: <driver> lists the table a reviewed change created
    Given connection "schema_<driver>" is registered for driver "<driver>"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_schema_<driver> (id INT)" and rollback "DROP TABLE uat_schema_<driver>"
    And I approve the change
    And I apply the change
    Then the schema lists table "uat_schema_<driver>"

    Examples:
      | driver    |
      | postgres  |
      | mysql     |
      | mariadb   |
      | cockroach |
      | sqlite    |

  Scenario Outline: <driver> does not invent a table
    Given connection "schema_absent_<driver>" is registered for driver "<driver>"
    Then the schema does not list table "uat_absent_<driver>"

    Examples:
      | driver    |
      | postgres  |
      | mysql     |
      | mariadb   |
      | cockroach |
      | sqlite    |

  Scenario Outline: <driver> has no schema route
    Then a GET to "/connections/demo/schema" for driver "<driver>" is refused naming the driver

    Examples:
      | driver        |
      | redis         |
      | valkey        |
      | dragonfly     |
      | mongo         |
      | couchdb       |
      | elasticsearch |
      | meilisearch   |

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/connections/anything/schema"
    Then the response status is 401
