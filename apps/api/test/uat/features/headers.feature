Feature: per-test headers, the bearer token and cookies
  The suite sets request headers per test: it can clear the default bearer, add
  a custom header, carry a cookie, and send a body verbatim. These are the seams
  a negative or contract scenario needs.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: a request without the bearer is rejected
    Given I clear the auth header
    When I read "SELECT 1 AS n"
    Then the request is rejected
    And the response status is 401

  Scenario: a custom header rides along
    Given I set header "X-UAT" to "yes"
    When I read "SELECT 1 AS n"
    Then the result has 1 row(s)
    And the first row's "n" is "1"

  Scenario: a cookie rides along
    Given I set cookie "uat" to "1"
    When I read "SELECT 1 AS n"
    Then the result has 1 row(s)

  Scenario: a raw payload is sent verbatim
    When I send a POST to "/external/v1/engine/sql/connections/demo/query" with payload:
      """
      {"sql": "SELECT 7 AS n"}
      """
    Then the result has 1 row(s)
    And the first row's "n" is "7"
