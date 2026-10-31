Feature: POST /external/v1/engine/{driver}/connections/{name}/flush
  Emptying the database needs its name typed back, and only then empties it.

  Background:
    Given the stack is running

  Scenario: a flush without the confirmation is refused
    Given connection "cache_flush_guard" is registered for driver "redis"
    And the redis key "flush:keep" holds "me"
    Then the redis flush is rejected

  Scenario: a confirmed flush empties the database
    Given connection "cache_flush" is registered for driver "redis"
    And the redis key "flush:gone" holds "x"
    When I empty the redis database as "cache_flush"
    Then the redis database is empty

  Scenario: the route needs a session
    Given no session
    When I send a POST to "/external/v1/engine/redis/connections/anything/flush" with body:
      """
      {"confirm":"anything"}
      """
    Then the response status is 401
