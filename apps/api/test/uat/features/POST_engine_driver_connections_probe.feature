Feature: POST /external/v1/engine/{driver}/connections/probe
  The registration form can test a target before it is saved.

  Background:
    Given the stack is running

  Scenario Outline: a reachable <driver> target answers
    Given connection "probe_<driver>" is registered for driver "<driver>"
    Then I probe the current target

    Examples:
      | driver   |
      | postgres |
      | mysql    |
      | redis    |

  Scenario: a closed port is refused
    Given connection "probe_closed" is registered for driver "postgres"
    Then probing a closed port is refused
