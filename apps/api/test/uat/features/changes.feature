Feature: changes — submit, validate, approve, apply
  As an operator I want every schema/data change validated against its own
  rollback before I approve it, so applying is never a leap of faith.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: a change is validated against its rollback
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_c (id INT, note TEXT); INSERT INTO uat_c VALUES (1, 'a;b')" and rollback "DROP TABLE uat_c"
    Then the change is "validated"

  Scenario: an invalid change is rejected without being recorded
    When I submit a change with query "INSERT INTO no_such_table VALUES (1)" and rollback "DELETE FROM no_such_table"
    Then the change is "failed"
    And the change was not recorded

  Scenario: an approved change applies and lands in the schema
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_w (id INT); INSERT INTO uat_w VALUES (7)" and rollback "DROP TABLE uat_w"
    Then the change is "validated"
    And I approve the change
    Then the change is "approved"
    And I apply the change
    Then the change is "applied"
    And the schema lists table "uat_w"
    And table "uat_w" has 1 row(s)

  Scenario: a rejected change is kept as an artifact
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_rej (id INT)" and rollback "DROP TABLE uat_rej"
    Then the change is "validated"
    And I reject the change
    Then the change is "rejected"
    And the change list still contains it

  Scenario: a rejected change never applies
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_rejected (id INT)" and rollback "DROP TABLE uat_rejected"
    Then the change is "validated"
    And I reject the change
    Then the change is "rejected"
    And applying is rejected
    And the schema does not list table "uat_rejected"

  Scenario: a qualified rollback is not applied to the live schema during validation
    When I run the write "CREATE TABLE IF NOT EXISTS uat_live_guard (id INT)"
    Then the schema lists table "uat_live_guard"
    When I submit a change with query "CREATE TABLE public.uat_probe_guard (id INT)" and rollback "DROP TABLE public.uat_live_guard"
    Then the change is "validated"
    And the schema lists table "uat_live_guard"

  Scenario: an applied change carries estimates and actuals
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_est (id INT)" and rollback "DROP TABLE uat_est"
    Then the change is "validated"
    And I approve the change
    And I apply the change
    Then the change is "applied"
    And the change records an estimated duration and affected count

  Scenario: a dry run and analyze stay available
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_a (v INT)" and rollback "DROP TABLE uat_a"
    Then the change is "validated"
    And I approve the change
    And I apply the change
    Then the change is "applied"
    And I dry-run "UPDATE uat_a SET v = 99"
    And analyze on table "uat_a" succeeds

  Scenario: a due scheduled change is applied by the tick
    When I submit a change scheduled at now with query "CREATE TABLE IF NOT EXISTS uat_sched (id INT)" and rollback "DROP TABLE uat_sched"
    Then the change is "validated"
    And I approve the change
    Then the change is "approved"
    And a scheduler tick applies the due change
    Then the change is "applied"
    And the schema lists table "uat_sched"
