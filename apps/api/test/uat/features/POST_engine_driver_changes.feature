Feature: POST /external/v1/engine/{driver}/changes
  A change is proven before it is recorded: its rollback runs against it inside a transaction that always rolls back.

  Background:
    Given the stack is running
    And a clean registry
    And connection "submit_pg" is registered for driver "postgres"

  Scenario: a write is validated in a scratch schema
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_submit (id INT)" and rollback "DROP TABLE uat_submit"
    Then the change is "validated"
    And the change was verified in a scratch schema

  Scenario: a change without a rollback is refused
    When I send a POST to "/external/v1/engine/postgres/changes" with body:
      """
      {"connection":"submit_pg","query":"CREATE TABLE uat_no_rollback (id INT)"}
      """
    Then the response status is 400

  Scenario: a rollback that does not undo the change is rejected
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_bad_rollback (id INT)" and rollback "SELECT 1"
    Then the change is "failed"

  Scenario: submitting needs a session
    Given no session
    When I send a POST to "/external/v1/engine/postgres/changes" with body:
      """
      {"connection":"submit_pg","query":"SELECT 1","rollback":"SELECT 1"}
      """
    Then the response status is 401
