Feature: POST /internal/callback/scheduler
  The external scheduler drives the tick; one tick applies what is due, whatever driver owns it.

  Background:
    Given the stack is running
    And a clean registry

  Scenario: one tick applies the due change of two drivers
    Given connection "tick_pg" is registered for driver "postgres"
    And a due change is scheduled on the current connection
    Given connection "tick_my" is registered for driver "mysql"
    And a due change is scheduled on the current connection
    When a scheduler tick runs
    Then the tick applied both scheduled changes

  Scenario: the callback needs its signature
    Given no session
    When I send a POST to "/internal/callback/scheduler"
    Then the response status is 401
