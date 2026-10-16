Feature: changes verify in a throwaway scratch schema
  As an operator I want a change proven against a disposable copy of the
  structures it touches, so a bad change cannot reach the live schema and
  MySQL-family DDL can be verified at last.

  Background:
    Given the stack is running

  Scenario: a postgres change verifies in a scratch schema and leaves none behind
    Given connection "pg_scratch" is registered
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_pg_scratch (id INT, note TEXT)" and rollback "DROP TABLE uat_pg_scratch"
    Then the change is "validated"
    And the change was verified in a scratch schema
    And no scratch schemas remain

  Scenario: a change against an existing table is verified on its structure
    Given connection "pg_scratch" is registered
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_pg_base (id INT, note TEXT)" and rollback "DROP TABLE uat_pg_base"
    Then the change is "validated"
    And I approve the change
    And I apply the change
    Then the change is "applied"
    When I submit a change with query "UPDATE uat_pg_base SET note = 'x' WHERE id = 1" and rollback "UPDATE uat_pg_base SET note = '' WHERE id = 1"
    Then the change is "validated"
    And no scratch schemas remain

  Scenario: a mysql-family DDL change verifies instead of being unverified
    Given connection "my_scratch" is registered for driver "mariadb"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_my_scratch (id INT, note VARCHAR(64))" and rollback "DROP TABLE uat_my_scratch"
    Then the change is "validated"
    And the change was verified in a scratch schema
    And no scratch schemas remain
