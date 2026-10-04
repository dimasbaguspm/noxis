Feature: Redis keys and metrics
  The /v1/redis namespace browses a live server: keys with their type and TTL,
  values rendered per type, INFO metrics, and key removal. Redis writes stop
  at deleting keys because a Redis value cannot be rolled back.

  Scenario: browse keys with their type and TTL
    Given the stack is running
    And a redis connection named "cache"
    And the redis server answers
    And the redis key "greeting" holds "hello"
    And the redis key "greeting" lives for 600 seconds
    And the redis key "queue" holds "a"
    When I browse redis keys matching "*"
    Then the browse finds the key "greeting"
    And the key "greeting" reports a ttl
    And I inspect the redis key "greeting"
    And the value grid shows "hello"

  Scenario: a list value renders every element
    Given the stack is running
    And a redis connection named "cache"
    And the redis list "queue" contains "a"
    When I inspect the redis key "queue"
    Then the value grid shows "a"

  Scenario: INFO reports the server
    Given the stack is running
    And a redis connection named "cache"
    When I read redis stats
    Then the redis stats name the server version
    And the redis stats report the "redis" flavour

  Scenario: a Valkey server is served by the same family
    Given the stack is running
    And a valkey connection named "valkey"
    And the redis server answers
    And the redis key "valkey:key" holds "from-valkey"
    When I browse redis keys matching "valkey:*"
    Then the browse finds the key "valkey:key"
    And I read redis stats
    And the redis stats report the "valkey" flavour

  Scenario: removing keys, and the guard on emptying the database
    Given the stack is running
    And a redis connection named "cache"
    And the redis key "temp:1" holds "x"
    And the redis key "temp:2" holds "y"
    And the redis key "keep" holds "me"
    When I delete the redis keys "temp:1,temp:2"
    And the redis flush is rejected
    Then I browse redis keys matching "*"
    And the browse finds the key "keep"
    When I empty the redis database as "cache"
    Then the redis database is empty
