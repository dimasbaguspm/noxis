Feature: GET /external/v1/me
  Every page starts by asking who the caller is.

  Background:
    Given the stack is running

  Scenario: the session identifies its account
    Then my profile names the root account

  Scenario: a session is required
    When a request to "/external/v1/me" without a token is refused
    Then the response status is 401
