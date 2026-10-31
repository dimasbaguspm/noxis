Feature: POST /external/v1/engine/{driver}/connections/{name}/keys/ttl
  An expiry is metadata and reversible, so it is set and cleared without a review.

  Background:
    Given the stack is running

  Scenario: an expiry is set and cleared
    Given connection "cache_ttl_edit" is registered for driver "redis"
    And the redis key "ttl:probe" holds "x"
    When I set the expiry of the redis key "ttl:probe" to 600 seconds
    Then the key "ttl:probe" reports a ttl
    When I clear the expiry of the redis key "ttl:probe"
    Then the key "ttl:probe" reports no ttl

  Scenario: the route needs a session
    Given no session
    When I send a POST to "/external/v1/engine/redis/connections/anything/keys/ttl" with body:
      """
      {"keys":["x"],"ttlMs":1000}
      """
    Then the response status is 401
