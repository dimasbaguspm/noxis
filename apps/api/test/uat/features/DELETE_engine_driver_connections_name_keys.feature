Feature: DELETE /external/v1/engine/{driver}/connections/{name}/keys
  Removing keys is the only write a cache offers: a value has no rollback, so it is never written.

  Background:
    Given the stack is running

  Scenario: the named keys are removed and the others stay
    Given connection "cache_del" is registered for driver "redis"
    And the redis key "del:1" holds "x"
    And the redis key "del:2" holds "y"
    And the redis key "del:keep" holds "me"
    When I delete the redis keys "del:1,del:2"
    Then I browse redis keys matching "del:*"
    And the browse finds the key "del:keep"
    And the browse does not find the key "del:1"

  Scenario: the route needs a session
    Given no session
    When I send a DELETE to "/external/v1/engine/redis/connections/anything/keys"
    Then the response status is 401
