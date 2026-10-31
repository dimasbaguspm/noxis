Feature: GET /external/v1/engine/{driver}/connections
  The console lists what is registered for one engine, and never a password.

  Background:
    Given the stack is running

  Scenario Outline: the <driver> list hides its passwords
    Given connection "list_<driver>" is registered for driver "<driver>"
    Then the connection list hides the password

    Examples:
      | driver        |
      | postgres      |
      | mysql         |
      | mariadb       |
      | cockroach     |
      | sqlite        |
      | redis         |
      | valkey        |
      | dragonfly     |
      | mongo         |
      | couchdb       |
      | elasticsearch |
      | meilisearch   |

  Scenario: the list needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/connections"
    Then the response status is 401
