Feature: GET /external/v1/engine/{driver}/changes
  The queue a reviewer works from; a rejected change stays in it as an artifact.

  Background:
    Given the stack is running
    And a clean registry
    And connection "queue_pg" is registered for driver "postgres"

  Scenario: a rejected change is still listed
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_queue_keep (id INT)" and rollback "DROP TABLE uat_queue_keep"
    Then the change is "validated"
    When I reject the change
    Then the change list still contains it

  Scenario: the queue needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/changes"
    Then the response status is 401
