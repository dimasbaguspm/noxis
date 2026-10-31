Feature: GET /external/v1/engine/{driver}/changes/{id}
  One change by id, which is what the review page loads.

  Background:
    Given the stack is running
    And a clean registry
    And connection "open_pg" is registered for driver "postgres"

  Scenario: the change opens
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_open (id INT)" and rollback "DROP TABLE uat_open"
    Then the change is "validated"
    And I open the change

  Scenario: an unknown id is refused
    When I send a GET to "/external/v1/engine/postgres/changes/00000000-0000-0000-0000-000000000000"
    Then the response status is 404

  Scenario: the detail needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/changes/00000000-0000-0000-0000-000000000000"
    Then the response status is 401
