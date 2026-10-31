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
    When I run the write "INSERT INTO uat_pg_base (id, note) VALUES (1, 'old')"
    When I submit a change with query "UPDATE uat_pg_base SET note = 'x' WHERE id = 1" and rollback "UPDATE uat_pg_base SET note = 'old' WHERE id = 1"
    Then the change is "validated"
    And no scratch schemas remain

  Scenario: a rollback that leaves changed data fails validation
    Given connection "pg_scratch" is registered
    When I run the write "CREATE TABLE IF NOT EXISTS uat_pg_restore (id INT PRIMARY KEY, note TEXT)"
    And I run the write "INSERT INTO uat_pg_restore (id, note) VALUES (1, 'old')"
    And I submit a change with query "UPDATE uat_pg_restore SET note = 'new' WHERE id = 1" and rollback "UPDATE uat_pg_restore SET note = 'new' WHERE id = 1"
    Then the change is "failed"
    And the change was not recorded
    And no scratch schemas remain

  Scenario: an identity column does not stop the scratch copy
    Given connection "pg_scratch" is registered
    When I run the write "CREATE TABLE IF NOT EXISTS uat_pg_identity (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, note TEXT)"
    And I run the write "INSERT INTO uat_pg_identity (note) VALUES ('old')"
    And I submit a change with query "UPDATE uat_pg_identity SET note = 'new' WHERE id = 1" and rollback "UPDATE uat_pg_identity SET note = 'old' WHERE id = 1"
    Then the change is "validated"
    And I submit a change with query "INSERT INTO uat_pg_identity (note) VALUES ('new')" and rollback "DELETE FROM uat_pg_identity WHERE note = 'new'"
    Then the change is "validated"
    And no scratch schemas remain

  Scenario: an empty table cannot prove a DML rollback
    Given connection "pg_scratch" is registered
    When I run the write "CREATE TABLE IF NOT EXISTS uat_pg_empty (id INT, note TEXT)"
    And I submit a change with query "UPDATE uat_pg_empty SET note = 'new'" and rollback "UPDATE uat_pg_empty SET note = 'old'"
    Then the change is unverified because "no-rows"
    And no scratch schemas remain

  Scenario: a mysql-family DDL change verifies instead of being unverified
    Given connection "my_scratch" is registered for driver "mariadb"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_my_scratch (id INT, note VARCHAR(64))" and rollback "DROP TABLE uat_my_scratch"
    Then the change is "validated"
    And the change was verified in a scratch schema
    And no scratch schemas remain

  Scenario Outline: a qualified DDL change is proved in the scratch database on <driver>
    Given connection "<driver>_qualified" is registered for driver "<driver>"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS demo.uat_qualified (id INT, note VARCHAR(64))" and rollback "DROP TABLE demo.uat_qualified"
    Then the change is "validated"
    And the change was verified in a scratch schema
    And no scratch schemas remain

    Examples:
      | driver  |
      | mysql   |
      | mariadb |

  Scenario: a qualified change against an existing table is proved on its structure
    Given connection "my_scratch" is registered for driver "mariadb"
    When I run the write "CREATE TABLE IF NOT EXISTS demo.uat_qualified_base (id INT, note VARCHAR(64))"
    And I run the write "INSERT INTO demo.uat_qualified_base (id, note) VALUES (1, 'old')"
    And I submit a change with query "ALTER TABLE demo.uat_qualified_base ADD COLUMN tag VARCHAR(32)" and rollback "ALTER TABLE demo.uat_qualified_base DROP COLUMN tag"
    Then the change is "validated"
    And the change was verified in a scratch schema
    And no scratch schemas remain

  Scenario: a qualified name in another database cannot be proved
    Given connection "my_scratch" is registered for driver "mariadb"
    When I submit a change with query "CREATE TABLE other.uat_foreign (id INT)" and rollback "DROP TABLE other.uat_foreign"
    Then the change is unverified because "foreign-schema"
    And no scratch schemas remain

  Scenario: a statement that changes the server is not dry-run
    Given connection "my_scratch" is registered for driver "mariadb"
    When I submit a change with query "CREATE USER uat_nobody" and rollback "DROP USER uat_nobody"
    Then the change is unverified because "server-ddl"
