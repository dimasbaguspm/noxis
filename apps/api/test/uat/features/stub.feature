@stub
Feature: a recorded run replays without the stack
  Tagged @stub, this feature runs against the real stack by default. Record it
  once with UAT_RECORD=1, then replay it offline — no docker — with UAT_STUB=1.
  The store lives under test/uat/stubs and is keyed by method and path.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: a read replays from the recorded store
    When I read "SELECT 41+1 AS n"
    Then the result has 1 row(s)
    And the first row's "n" is "42"
