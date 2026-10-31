Feature: GET /external/v1/engine/{driver}/connections/{name}/keys
  Browse a keyspace by pattern, with each key's type and TTL.

  Background:
    Given the stack is running

  Scenario Outline: <driver> browses its own keyspace
    Given connection "cache_<driver>" is registered for driver "<driver>"
    And the redis key "<driver>:key" holds "v"
    When I browse redis keys matching "<driver>:*"
    Then the browse finds the key "<driver>:key"

    Examples:
      | driver    |
      | redis     |
      | valkey    |
      | dragonfly |

  Scenario: a key carries its type and its ttl
    Given connection "cache_ttl" is registered for driver "redis"
    And the redis key "keys:greeting" holds "hello"
    And the redis key "keys:greeting" lives for 600 seconds
    When I browse redis keys matching "keys:*"
    Then the browse finds the key "keys:greeting"
    And the key "keys:greeting" reports a ttl

  Scenario: a pattern that matches nothing finds nothing
    Given connection "cache_miss" is registered for driver "redis"
    When I browse redis keys matching "absent:*"
    Then the browse does not find the key "absent:key"

  Scenario: a keyspace is walked past one page
    Given connection "cache_walk" is registered for driver "redis"
    And the redis key "walk:1" holds "a"
    And the redis key "walk:2" holds "b"
    And the redis key "walk:3" holds "c"
    When I walk every redis key matching "walk:*" in pages of 1
    Then the walk found 3 key(s) with no cursor left

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/redis/connections/anything/keys"
    Then the response status is 401
