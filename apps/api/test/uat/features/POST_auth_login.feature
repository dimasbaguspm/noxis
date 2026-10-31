Feature: POST /external/v1/auth/login
  A local account signs in with an email and a password, and nothing else mints a session.

  Background:
    Given the stack is running

  Scenario: the root account signs in
    When I log in with email "root@noxis.local" and password "noxis-admin"
    Then the login succeeds

  Scenario: a wrong password mints nothing
    When I log in with email "root@noxis.local" and password "wrong-password"
    Then the login is refused
    And the response status is 401

  Scenario: an unknown account mints nothing
    When I log in with email "nobody@example.com" and password "wrong-password"
    Then the login is refused
    And the response status is 401
