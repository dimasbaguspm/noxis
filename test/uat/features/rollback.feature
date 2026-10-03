Feature: doorman rollback — change previews that never persist
  As an operator I want to validate dangerous DDL as a dry run, so nothing
  applies until I say so.

  Background:
    Given the stack is running
    And node "zeus" is registered
    And the master assigns engine "postgres" version "16-alpine" to node "zeus"
    And I alter engine "postgres" with "CREATE TABLE IF NOT EXISTS uat_r (keep INT)"

  Scenario: a drop previewed as a dry run leaves the column alone
    When I rollback engine "postgres" with "ALTER TABLE uat_r DROP COLUMN keep"
    Then the dry-run reports success
    And column "keep" still exists in table "uat_r" on engine "postgres"

  Scenario: a failing change is reported without being applied
    When I rollback engine "postgres" with "ALTER TABLE uat_r DROP COLUMN nope"
    Then the dry-run reports an error
    And column "keep" still exists in table "uat_r" on engine "postgres"