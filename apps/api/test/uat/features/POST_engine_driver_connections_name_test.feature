Feature: POST /external/v1/engine/{driver}/connections/{name}/test
  A registered connection answers for its own engine, and an unreachable one does not.

  Background:
    Given the stack is running

  Scenario: a reachable target passes
    Given connection "test_pg" is registered for driver "postgres"
    Then the connection test passes

  Scenario: an unreachable target fails
    Given connection "unreachable_pg" is registered for driver "postgres" on host "127.0.0.1" port 1
    Then the connection test fails
