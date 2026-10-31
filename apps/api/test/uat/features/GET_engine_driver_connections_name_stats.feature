Feature: GET /external/v1/engine/{driver}/connections/{name}/stats
  What the cache server says about itself: memory, clients and the keyspace.

  Background:
    Given the stack is running

  Scenario Outline: the <driver> server reports what it holds
    Given connection "stats_<driver>" is registered for driver "<driver>"
    And the redis key "stats:<driver>" holds "x"
    When I read the cache server's stats
    Then the cache stats report a used memory and at least one key

    Examples:
      | driver |
      | redis  |
      | valkey |

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/redis/connections/anything/stats"
    Then the response status is 401
