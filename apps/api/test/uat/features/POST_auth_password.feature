Feature: POST /external/v1/auth/password
  An account rotates its own password, and the old one stops working.

  Background:
    Given the stack is running

  Scenario: the owner rotates the password and rotates it back
    When I change my password to "uat-rotated-password" and back
    Then the response status is 200

  Scenario: a wrong old password changes nothing
    When changing my password with the wrong old password is refused
    Then the response status is 401
