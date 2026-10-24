@mock
Feature: the cache protocol mock records target traffic
  Redis and Valkey speak the same protocol, so one mock proves the commands
  noXis sends and how the replies become a browse and a value grid.

  Background:
    Given the stack is running

  Scenario Outline: a browse and a value read reach the mock on <driver>
    Given a mock <driver> connection named "mock_<driver>"
    And the mock cache holds the keys "greeting,queue" with value "hello"
    When I browse redis keys matching "*"
    Then the browse finds the key "greeting"
    And the mock cache received the command "SCAN"
    When I inspect the redis key "greeting"
    And the value grid shows "hello"
    And the mock cache received the command "GET"

    Examples:
      | driver |
      | redis  |
      | valkey |

  Scenario: a target error reaches the caller
    Given a mock redis connection named "mock_cache_error"
    And the mock cache answers with error "no such key"
    When I try to browse redis keys matching "*"
    Then the request is rejected
