Feature: GET /external/v1/users/{subject}
  One account, addressed by its subject.

  Background:
    Given the stack is running

  Scenario: the root profile carries its email
    Then the root profile carries its email

  Scenario: an unknown subject is refused
    When I send a GET to "/external/v1/users/local:nobody@example.com"
    Then the response status is 404

  Scenario: the profile needs a session
    Given no session
    When I send a GET to "/external/v1/users/local:nobody@example.com"
    Then the response status is 401
