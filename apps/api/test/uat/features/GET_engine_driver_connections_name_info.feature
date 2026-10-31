Feature: GET /external/v1/engine/{driver}/connections/{name}/info
  What the connected server says about itself, and what that profile can prove.

  Background:
    Given the stack is running

  Scenario Outline: the <driver> server reports its version and capabilities
    Given connection "info_<driver>" is registered for driver "<driver>"
    Then the server reports its version and capabilities

    Examples:
      | driver    |
      | postgres  |
      | mysql     |
      | mariadb   |
      | cockroach |
      | sqlite    |

  Scenario Outline: <driver> has no server info route
    Then a GET to "/connections/demo/info" for driver "<driver>" is refused naming the driver

    Examples:
      | driver        |
      | redis         |
      | mongo         |
      | couchdb       |
      | elasticsearch |

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/postgres/connections/anything/info"
    Then the response status is 401
