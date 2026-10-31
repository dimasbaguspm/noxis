Feature: POST /external/v1/engine/{driver}/changes/{id}/schedule
  A change can carry a slot, and the external scheduler applies what is due.

  Background:
    Given the stack is running
    And a clean registry
    And connection "schedule_pg" is registered for driver "postgres"

  Scenario: a change is scheduled and the tick applies it
    When I submit a change scheduled at now with query "CREATE TABLE IF NOT EXISTS uat_sched (id INT)" and rollback "DROP TABLE uat_sched"
    And I approve the change
    When a scheduler tick applies the due change
    Then the change is "applied"
