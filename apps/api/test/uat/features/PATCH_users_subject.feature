Feature: PATCH /external/v1/users/{subject}
  Role and status are managed by someone else, never by the account itself.

  Background:
    Given the stack is running

  Scenario: an operator cannot change their own account
    Then changing my own account is refused

  Scenario: an unknown subject is refused
    When I send a PATCH to "/external/v1/users/local:nobody@example.com" with body:
      """
      {"role":"admin"}
      """
    Then the response status is 404

  Scenario: the patch needs a session
    Given no session
    When I send a PATCH to "/external/v1/users/local:nobody@example.com" with body:
      """
      {"role":"admin"}
      """
    Then the response status is 401
