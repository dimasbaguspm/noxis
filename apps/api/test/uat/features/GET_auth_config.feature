Feature: GET /external/v1/auth/config
  The console reads its sign-in options before it holds a token, and the answer never carries a secret.

  Background:
    Given the stack is running

  Scenario: local sign-in is advertised
    Then the auth config is readable
    And the response matches snapshot "auth/config"

  Scenario: it is served without a token
    Then the auth config answers without a token
