Feature: GET /external/v1/engine/{driver}/connections/{name}/value
  One key rendered as a grid, per its type.

  Background:
    Given the stack is running

  Scenario: a string renders as one value
    Given connection "cache_val" is registered for driver "redis"
    And the redis key "val:string" holds "hello"
    When I inspect the redis key "val:string"
    Then the value grid shows "hello"
    And the response matches snapshot "memcache/value"

  Scenario: a list renders every element
    Given connection "cache_list" is registered for driver "redis"
    And the redis list "val:queue" contains "a"
    When I inspect the redis key "val:queue"
    Then the value grid shows "a"

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/redis/connections/anything/value?key=x"
    Then the response status is 401
