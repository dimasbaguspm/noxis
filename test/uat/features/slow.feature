@slow
Feature: Talking to a real engine
  The @fast suite proves refusals and shapes; this one proves the engine work against containers,
  because a fake cannot disagree with you. Run it with UAT_SLOW=1.

  Scenario: the probe measures the container
    Given I hold the ops token
    And I register the "mysql" container as "uat-engine-mysql"
    When I probe the target I registered
    Then the request succeeds
    And the capabilities name the "mysql" engine
    And the capabilities report a version

  Scenario: a read comes back from the container
    Given I hold the ops token
    And I register the "mariadb" container as "uat-engine-mariadb"
    When I run the statement "SELECT 1 AS one"
    Then the request succeeds
    And the result has 1 row

  Scenario: a write is still refused against a reachable engine
    Given I hold the ops token
    And I register the "mysql" container as "uat-engine-refuse"
    When I run the statement "DELETE FROM users"
    Then the request fails with status 400
    And the refusal mentions "delete"
