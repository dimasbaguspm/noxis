Feature: a batch of .sql files becomes changes sharing a batch id
  As an operator I want to upload a handful of migrations at once and see each
  one verified on its own, so a bad file never blocks the good ones.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: files are verified per file and share a batch id
    When I submit a batch of SQL files:
      | name    | query                                                        | rollback          |
      | 001.sql | CREATE TABLE IF NOT EXISTS uat_b1 (id INT)                   | DROP TABLE uat_b1 |
      | 002.sql | CREATE TABLE IF NOT EXISTS uat_b2 (id INT); DROP TABLE uat_b2 | DROP TABLE uat_b2 |
    Then file "001.sql" is verified in the batch
    And file "002.sql" is rejected in the batch with "one statement"
