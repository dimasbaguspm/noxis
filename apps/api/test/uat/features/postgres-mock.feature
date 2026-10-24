@mock
Feature: a protocol mock proves what noXis sends to a target
  A target-side mock records the requests noXis makes and answers with a
  controlled reply, so the client side is proven without a real database.

  Background:
    Given the stack is running

  Scenario: a query reaches the mock and its reply comes back
    Given a mock postgres connection named "mock_pg"
    And the mock postgres answers rows with column "n" and value "42"
    When I read "SELECT 42 AS n"
    Then the result has 1 row(s)
    And the first row's "n" is "42"
    And the mock postgres received the query "SELECT 42 AS n"

  Scenario: a connection test pings the mock target
    Given a mock postgres connection named "mock_pg_ping"
    Then the connection test passes
    And the mock postgres received a request

  Scenario: a target error reaches the caller
    Given a mock postgres connection named "mock_pg_error"
    And the mock postgres answers with error "relation missing does not exist"
    When I read "SELECT * FROM missing"
    Then the request is rejected
