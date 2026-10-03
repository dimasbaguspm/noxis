Feature: node lifecycle
  As an operator I want to hand a machine to noxis and get running databases
  back, so engines converge without manual docker work.

  Background:
    Given the stack is running
    And node "zeus" is registered

  Scenario: a healthy node shows up as up
    Then the node "zeus" status is "up"

  Scenario: assigning an engine spins up its database
    When the master assigns engine "postgres" version "16-alpine" to node "zeus"
    Then the instance "zeus-postgres" container is running image "postgres:16-alpine"
    And node "zeus" status detail mentions "created"

  Scenario: re-assigning an unchanged engine reports in sync
    Given the master assigns engine "postgres" version "16-alpine" to node "zeus"
    When the master assigns engine "postgres" version "16-alpine" to node "zeus"
    Then node "zeus" status detail mentions "in sync"

  Scenario: assigning to an unregistered node is rejected
    When the master tries to assign engine "postgres" version "16-alpine" to node "ghost"
    Then the master rejects the assignment