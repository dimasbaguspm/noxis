Feature: changes — submit, validate, approve, apply
  As an operator I want every schema/data change validated against its own
  rollback before I approve it, so applying is never a leap of faith.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: a change is validated against its rollback
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_c (id INT, note TEXT); INSERT INTO uat_c VALUES (1, 'a;b')" and rollback "DROP TABLE uat_c"
    Then the change is "validated"

  Scenario: an invalid change fails validation and cannot be approved
    When I submit a change with query "INSERT INTO no_such_table VALUES (1)" and rollback "DELETE FROM no_such_table"
    Then the change is "failed"
    And I approve the change
    Then applying is rejected

  Scenario: an approved change applies and lands in the schema
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_w (id INT); INSERT INTO uat_w VALUES (7)" and rollback "DROP TABLE uat_w"
    Then the change is "validated"
    And I approve the change
    Then the change is "approved"
    And I apply the change
    Then the change is "applied"
    And the schema lists table "uat_w"
    And table "uat_w" has 1 row(s)

  Scenario: a dry run and analytics stay available
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_a (v INT)" and rollback "DROP TABLE uat_a"
    Then the change is "validated"
    And I approve the change
    And I apply the change
    Then the change is "applied"
    And I dry-run "UPDATE uat_a SET v = 99"
    And analytics on table "uat_a" succeeds
