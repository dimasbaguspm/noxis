Feature: reads run directly, writes are guarded
  As an operator I want to ask questions without risk, and be stopped from
  changing data through the read path.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: a select returns its grid
    When I read "SELECT 6*6 AS n"
    Then the result has 1 row(s)
    And the first row's "n" is "36"

  Scenario: DML is rejected by the read-only guard
    When I read "INSERT INTO items VALUES (9, 'x')"
    Then the request is rejected

  Scenario: DDL smuggled behind a read is rejected
    When I read "SELECT 1; DROP TABLE items"
    Then the request is rejected
