Feature: POST /external/v1/auth/logout
  Signing out revokes the session it was called with, and only that one.

  Background:
    Given the stack is running

  Scenario: a session signs out and stops answering
    When I sign out and sign in again
    Then the response status is 200

  Scenario: signing out needs a session
    When a POST to "/external/v1/auth/logout" without a token is refused
    Then the response status is 401
