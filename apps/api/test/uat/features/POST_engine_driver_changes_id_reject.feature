Feature: POST /external/v1/engine/{driver}/changes/{id}/reject
  Closing a change without running it, and keeping it as an artifact.

  Background:
    Given the stack is running
    And a clean registry
    And connection "reject_pg" is registered for driver "postgres"

  Scenario: a change is rejected and never runs
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_reject (id INT)" and rollback "DROP TABLE uat_reject"
    And I reject the change
    Then the change is "rejected"
    And the schema does not list table "uat_reject"
