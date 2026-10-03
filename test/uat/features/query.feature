Feature: doorman query — read-only
  As an operator I want to ask questions of assigned databases without ever
  changing them (schema or data — writes live on alter).

  Background:
    Given the stack is running
    And node "zeus" is registered

  Scenario: a bare select returns its value
    Given the master assigns engine "postgres" version "16-alpine" to node "zeus"
    When I query engine "postgres" with "SELECT 6*6"
    Then the output is "36"

  Scenario: reads and writes split across ops
    Given the master assigns engine "postgres" version "16-alpine" to node "zeus"
    And I alter engine "postgres" with "CREATE TABLE IF NOT EXISTS uat_q (n INT)"
    And I alter engine "postgres" with "INSERT INTO uat_q VALUES (7), (8)"
    When I query engine "postgres" with "SELECT SUM(n) FROM uat_q"
    Then the output is "15"

  Scenario: DML is rejected by the read-only query guard
    Given the master assigns engine "postgres" version "16-alpine" to node "zeus"
    When I query engine "postgres" with "INSERT INTO uat_q VALUES (1)"
    Then the master rejects the request with a 400

  Scenario: DDL is rejected by the query guard
    Given the master assigns engine "postgres" version "16-alpine" to node "zeus"
    When I query engine "postgres" with "DROP TABLE uat_q"
    Then the master rejects the request with a 400

  Scenario: DDL smuggled after a read is rejected too
    Given the master assigns engine "postgres" version "16-alpine" to node "zeus"
    When I query engine "postgres" with "SELECT 1; DROP TABLE uat_q"
    Then the master rejects the request with a 400

  Scenario: a semicolon inside a string value does not split the batch
    Given the master assigns engine "postgres" version "16-alpine" to node "zeus"
    And I alter engine "postgres" with "CREATE TABLE IF NOT EXISTS uat_semi (v TEXT)"
    When I alter engine "postgres" with "INSERT INTO uat_semi VALUES ('a;b')"
    And I query engine "postgres" with "SELECT v FROM uat_semi"
    Then the output is "a;b"