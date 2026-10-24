Feature: SQL direct operations and the change artifact routes
  A write can be run directly (alter), and a reviewed change can be opened and
  scheduled by id, so every SQL route has a representative flow.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: a direct write creates a table and the browser sees it
    When I run the write "CREATE TABLE IF NOT EXISTS uat_direct (id INT)"
    Then the schema lists table "uat_direct"
    And table "uat_direct" has 0 row(s)

  Scenario: an unknown console operation is refused
    When I send an unknown console operation
    Then the request is refused as a bad request

  Scenario: a change can be opened and scheduled by id
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_sched_route (id INT)" and rollback "DROP TABLE uat_sched_route"
    Then the change is "validated"
    And I open the change
    When I approve the change
    And I schedule the change for now
    Then the change is scheduled

