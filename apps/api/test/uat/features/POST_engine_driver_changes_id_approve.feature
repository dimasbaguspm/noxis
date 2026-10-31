Feature: POST /external/v1/engine/{driver}/changes/{id}/approve
  Signing off a validated change, which the review page then offers to apply.

  Background:
    Given the stack is running
    And a clean registry
    And connection "approve_pg" is registered for driver "postgres"

  Scenario: a validated change is approved
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_approve (id INT)" and rollback "DROP TABLE uat_approve"
    Then the change is "validated"
    When I approve the change
    Then the change is "approved"

  Scenario: approving an unknown change is refused
    When I send a POST to "/external/v1/engine/postgres/changes/00000000-0000-0000-0000-000000000000/approve"
    Then the response status is 404
