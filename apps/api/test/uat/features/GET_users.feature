Feature: GET /external/v1/users
  The account directory, readable by an admin and free of secrets.

  Background:
    Given the stack is running

  Scenario: the directory lists the root account
    Then the user directory contains the root account

  Scenario: the directory needs a session
    Given no session
    When I send a GET to "/external/v1/users"
    Then the response status is 401
