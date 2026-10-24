@mock
Feature: the MongoDB protocol mock records target traffic
  A mock target speaks the driver's protocol, so a scenario proves the command
  noXis sent and how its reply became a result grid.

  Background:
    Given the stack is running

  Scenario: a find reaches the mock and its documents come back
    Given a mock mongo connection named "mock_mongo"
    And the mock mongo answers documents [{"sku": "a"}, {"sku": "b"}]
    When I run a mongo "find" request on "items" with filter {"sku": "a"} and value {}
    Then the mongo result has 2 row(s)
    And the mock mongo received a "find" command
    And the mock mongo received a find filter with "sku" = "a"

  Scenario: a target error reaches the caller
    Given a mock mongo connection named "mock_mongo_error"
    And the mock mongo answers with error "collection missing does not exist"
    When I run a mongo "find" request on "missing" with filter {} and value {}
    Then the mongo request is rejected with "collection missing"
