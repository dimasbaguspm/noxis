@drivers
Feature: the driver catalog states what each driver can do
  As an operator I want the console to offer a section only where the driver can
  serve it, so a capability the module does not implement is refused by name
  instead of failing obscurely.

  Background:
    Given the stack is running

  Scenario Outline: <driver> advertises exactly what its module serves
    Then driver "<driver>" advertises exactly "<capabilities>"

    Examples:
      | driver    | capabilities |
      | postgres  | sql          |
      | mysql     | sql          |
      | mariadb   | sql          |
      | sqlite    | sql          |
      | cockroach | sql          |
      | redis     | keys         |
      | valkey    | keys         |
      | dragonfly | keys         |
      | mongo     | documents    |
      | couchdb   | documents    |
      | elasticsearch | search   |

  Scenario Outline: a capability a driver lacks is refused by name
    Then a <method> to "<path>" for driver "<driver>" is refused naming the driver

    Examples:
      | method | path                       | driver   |
      | POST   | /changes                   | redis    |
      | GET    | /connections/demo/keys     | postgres |
      | POST   | /connections/demo/requests | mysql    |

  Scenario: one tick applies the due change of every driver
    Given connection "tick_pg" is registered for driver "postgres"
    And a due change is scheduled on the current connection
    Given connection "tick_my" is registered for driver "mysql"
    And a due change is scheduled on the current connection
    When a scheduler tick runs
    Then the tick applied both scheduled changes
