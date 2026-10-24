@mock
Feature: the MySQL and MariaDB protocol mocks record target traffic
  Both MySQL-family drivers speak the same protocol, so one mock proves the
  request noXis sends and the reply it maps back into a result.

  Background:
    Given the stack is running

  Scenario Outline: a query reaches the mock and its reply comes back on <driver>
    Given a mock <driver> connection named "mock_<driver>"
    And the mock mysql answers rows with column "n" and value "42"
    When I read "SELECT 42 AS n"
    Then the result has 1 row(s)
    And the first row's "n" is "42"
    And the mock mysql received the query "SELECT 42 AS n"

    Examples:
      | driver  |
      | mysql   |
      | mariadb |

  Scenario: a target error reaches the caller on mysql
    Given a mock mysql connection named "mock_mysql_error"
    And the mock mysql answers with error "table missing does not exist"
    When I read "SELECT * FROM missing"
    Then the request is rejected
