Feature: doorman analytics — table statistics
  As an operator I want to refresh planner statistics for a table, so queries
  on a grown dataset stay fast.

  Background:
    Given the stack is running
    And node "zeus" is registered
    And the master assigns engine "postgres" version "16-alpine" to node "zeus"

  Scenario: ANALYZE runs on the requested table
    Given I alter engine "postgres" with "CREATE TABLE IF NOT EXISTS uat_s (v INT)"
    When I analyze engine "postgres" on table "uat_s"
    Then the analyze succeeds