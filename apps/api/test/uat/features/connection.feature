Feature: connections
  As an operator I want to register the databases noxis can reach, so the
  console works against databases that already exist.

  Background:
    Given the stack is running

  Scenario: a registered connection answers and hides its password
    Given connection "demo" is registered
    Then the connection test passes
    And the connection list hides the password
    And the response matches snapshot "sql/connection-list"
    And the operator console is served at "/"
