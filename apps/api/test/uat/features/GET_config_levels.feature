Feature: GET /external/v1/config/levels
  The console renders its grant picker from the levels the server enforces.

  Background:
    Given the stack is running

  Scenario Outline: <level> is offered
    Then the level catalog lists "<level>"

    Examples:
      | level |
      | read  |
      | write |
      | admin |

  Scenario: the catalog needs a session
    When a request to "/external/v1/config/levels" without a token is refused
    Then the response status is 401
