Feature: authentication and configuration
  The console bootstraps from public configuration, and every other route needs a
  session; bad credentials never mint one.

  Background:
    Given the stack is running

  Scenario: the public catalogs are served
    Then the driver catalog lists "postgres"
    And the level catalog lists "read"
    And the auth config is readable
    And the response matches snapshot "auth/config"

  Scenario: bad credentials are refused
    When I log in with email "root@noxis.local" and password "wrong-password"
    Then the login is refused

  Scenario: a session identifies its account
    Then my profile names the root account

  Scenario: protected routes require a token
    When a request to "/external/v1/me" without a token is refused
    Then the response status is 401
