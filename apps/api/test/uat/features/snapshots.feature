Feature: responses match committed snapshots
  A scenario pins the wire contract of a response: its scrubbed body is compared
  against a committed golden, so a field rename or a shape change shows up as a
  diff in review. Volatile fields — ids, timestamps, durations and scratch schema
  names — are masked before the comparison.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: a validated change matches its golden
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_snap (id INT)" and rollback "DROP TABLE uat_snap"
    Then the change is "validated"
    And the response matches snapshot "sql/change-validated"

  Scenario: a read result matches its golden
    When I read "SELECT 6*6 AS n"
    Then the result has 1 row(s)
    And the response matches snapshot "sql/select"
