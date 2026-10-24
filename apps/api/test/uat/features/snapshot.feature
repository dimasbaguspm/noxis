@snapshot
Feature: snapshot and restore a target database
  A target can be snapshotted and put back with its own native dump, so a
  scenario can mutate freely and observe the original. Snapshots live in memory
  for the run; nothing is written to the repository.

  Background:
    Given the stack is running

  Scenario: a postgres snapshot restores the row count
    Given connection "demo" is registered
    And I submit a change with query "CREATE TABLE IF NOT EXISTS uat_snapdb (id INT); INSERT INTO uat_snapdb VALUES (1)" and rollback "DROP TABLE uat_snapdb"
    And the change is "validated"
    And I approve the change
    And I apply the change
    And the database "demo" is snapshotted as "clean"
    And table "uat_snapdb" has 1 row(s)
    When I submit a change with query "INSERT INTO uat_snapdb VALUES (2)" and rollback "DELETE FROM uat_snapdb WHERE id = 2"
    And I approve the change
    And I apply the change
    Then table "uat_snapdb" has 2 row(s)
    When the "clean" snapshot of "demo" is restored
    Then table "uat_snapdb" has 1 row(s)

  Scenario Outline: a mysql-family target snapshots and restores on <driver>
    Given connection "snap_<driver>" is registered for driver "<driver>"
    And I submit a change with query "CREATE TABLE IF NOT EXISTS uat_mysnap (id INT); INSERT INTO uat_mysnap VALUES (1)" and rollback "DROP TABLE uat_mysnap"
    And the change is "validated"
    And I approve the change
    And I apply the change
    And the database "snap_<driver>" is snapshotted as "clean"
    When I submit a change with query "INSERT INTO uat_mysnap VALUES (2)" and rollback "DELETE FROM uat_mysnap WHERE id = 2"
    And I approve the change
    And I apply the change
    When I read "SELECT COUNT(*) AS n FROM uat_mysnap"
    Then the first row's "n" is "2"
    When the "clean" snapshot of "snap_<driver>" is restored
    And I read "SELECT COUNT(*) AS n FROM uat_mysnap"
    Then the first row's "n" is "1"

    Examples:
      | driver  |
      | mariadb |
      | mysql   |

  Scenario: a mongo target snapshots and restores its documents
    Given a mongo connection named "snap_docs"
    And the mongo server answers
    And I seed mongo collection "snaps" with [{"sku": "a"}]
    And the database "snap_docs" is snapshotted as "clean"
    When I seed mongo collection "snaps" with [{"sku": "b"}]
    And I run a mongo "find" request on "snaps" with filter {} and value {}
    Then the mongo result has 2 row(s)
    When the "clean" snapshot of "snap_docs" is restored
    And I run a mongo "find" request on "snaps" with filter {} and value {}
    Then the mongo result has 1 row(s)

  Scenario Outline: a cache target snapshots and restores on <driver>
    Given a <driver> connection named "snap_<driver>"
    And the redis key "snap:<driver>:keep" holds "v1"
    And the database "snap_<driver>" is snapshotted as "clean"
    And the redis key "snap:<driver>:new" holds "v2"
    When I browse redis keys matching "snap:<driver>:*"
    Then the browse finds the key "snap:<driver>:new"
    When the "clean" snapshot of "snap_<driver>" is restored
    And I browse redis keys matching "snap:<driver>:*"
    Then the browse does not find the key "snap:<driver>:new"

    Examples:
      | driver |
      | redis  |
      | valkey |
