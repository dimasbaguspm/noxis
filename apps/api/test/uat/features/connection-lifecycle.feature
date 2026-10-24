Feature: connection and instance lifecycle
  A connection is registered, tested, found by its stable id across families,
  locked to its identity, and removed; rotating a credential never leaks it.

  Background:
    Given the stack is running

  Scenario: a connection is registered, inspected and deleted
    Given a fresh connection "lifecycle" is registered
    Then the connection test passes
    And the instances list contains the current connection
    And I open the current instance
    When I delete the current connection
    Then the connection is gone

  Scenario: a connection's identity is locked after registration
    Given a fresh connection "locked" is registered
    Then changing the connection's host is refused

  Scenario: rotating an instance password keeps the secret off the wire
    Given a fresh connection "rotate" is registered
    And the instances list contains the current connection
    When I rotate the instance password
    Then the current connection still hides its password
