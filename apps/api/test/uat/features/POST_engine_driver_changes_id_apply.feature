Feature: POST /external/v1/engine/{driver}/changes/{id}/apply
  The one place a write reaches the database, and only after a review.

  Background:
    Given the stack is running
    And a clean registry
    And connection "apply_pg" is registered for driver "postgres"

  Scenario: an approved change is applied
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_apply (id INT)" and rollback "DROP TABLE uat_apply"
    And I approve the change
    When I apply the change
    Then the change is "applied"
    And the schema lists table "uat_apply"

  Scenario: applying before approval is rejected
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_unapproved (id INT)" and rollback "DROP TABLE uat_unapproved"
    Then the change is "validated"
    When applying an unapproved change is refused
    Then the response status is 400
