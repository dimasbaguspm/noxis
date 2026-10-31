Feature: POST /external/v1/instances/{id}/password
  An instance's stored password is rotated by an admin, and the console still never reads it back.

  Background:
    Given the stack is running

  Scenario: the password rotates and stays hidden
    Given a clean registry
    And connection "rotate_pg" is registered for driver "postgres"
    And the instances list contains the current connection
    When I rotate the instance password
    Then the current connection still hides its password

  Scenario: an unknown id is refused
    Then rotating an unknown instance password is refused
