Feature: doorman alter — DDL only
  As an operator I want to change schemas on an explicit DDL path, keeping
  reads and DML out of it.

  Background:
    Given the stack is running
    And node "zeus" is registered
    And the master assigns engine "postgres" version "16-alpine" to node "zeus"

  Scenario: an alter applies DDL and the change sticks
    When I alter engine "postgres" with "CREATE TABLE IF NOT EXISTS uat_a (n INT, m INT)"
    And I alter engine "postgres" with "ALTER TABLE uat_a ADD COLUMN k INT"
    And I query engine "postgres" with "SELECT k FROM uat_a WHERE n = 1 AND m = 2"
    Then the output is ""

  Scenario: reads are rejected by the alter guard
    When I alter engine "postgres" with "SELECT 1"
    Then the master rejects the request with a 400