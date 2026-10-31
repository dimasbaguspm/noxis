Feature: GET /external/v1/config/members
  An admin picks who to grant from the account directory, which never carries a secret.

  Background:
    Given the stack is running

  Scenario: the directory lists the root account without secrets
    Then the member directory lists the root account

  Scenario: the directory needs a session
    When a request to "/external/v1/config/members" without a token is refused
    Then the response status is 401
