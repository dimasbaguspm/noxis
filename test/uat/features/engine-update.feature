@slow
Feature: engine updates — data follows the major, majors start fresh
  As an operator I want to upgrade a database engine without losing data,
  while major bumps get a clean cluster (migration stays a human job).

  Background:
    Given the stack is running
    And node "zeus" is registered

  Scenario: a same-major update keeps the data volume
    Given the master assigns engine "postgres" version "16-alpine" to node "zeus"
    And I alter engine "postgres" with "CREATE TABLE IF NOT EXISTS uat_k (v INT)"
    And I alter engine "postgres" with "INSERT INTO uat_k VALUES (42)"
    When the master assigns engine "postgres" version "16.6-alpine" to node "zeus"
    Then the instance "zeus-postgres" container is running image "postgres:16.6-alpine"
    And I query engine "postgres" with "SELECT v FROM uat_k"
    Then the output is "42"

  Scenario: a major bump starts fresh, the old cluster stays intact
    Given the master assigns engine "postgres" version "16.6-alpine" to node "zeus"
    And I alter engine "postgres" with "CREATE TABLE IF NOT EXISTS uat_m (v INT)"
    And I alter engine "postgres" with "INSERT INTO uat_m VALUES (42)"
    When the master assigns engine "postgres" version "17-alpine" to node "zeus"
    Then the instance "zeus-postgres" container is running image "postgres:17-alpine"
    And I query engine "postgres" with "SELECT count(*) FROM information_schema.tables WHERE table_name='uat_m'"
    Then the output is "0"