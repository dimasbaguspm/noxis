Feature: GET /external/v1/config/drivers
  The catalog is derived from the registered modules, so the console can only offer what the process can serve.

  Background:
    Given the stack is running

  Scenario Outline: <driver> is listed with what it serves
    Then the driver catalog lists "<driver>"
    And driver "<driver>" advertises exactly "<capabilities>"

    Examples:
      | driver        | capabilities |
      | postgres      | sql          |
      | mysql         | sql          |
      | mariadb       | sql          |
      | sqlite        | sql          |
      | cockroach     | sql          |
      | redis         | keys         |
      | valkey        | keys         |
      | dragonfly     | keys         |
      | mongo         | documents    |
      | couchdb       | documents    |
      | elasticsearch | search       |
      | meilisearch   | search       |
      | splunk        | search       |

  Scenario: a driver no module serves is not listed
    Then the driver catalog does not list "oracle"

  Scenario: the catalog needs a session
    When a request to "/external/v1/config/drivers" without a token is refused
    Then the response status is 401
