Feature: POST /external/v1/access-requests/{id}/reject
  Rejection follows the same rule: the requester cannot close their own ask.

  Background:
    Given the stack is running
    And a clean registry
    And connection "reject_pg" is registered for driver "postgres"

  Scenario: the requester cannot reject their own request
    When I request the "write" level on the current connection
    Then the access request list contains it
    When I reject the access request
    Then the request is refused as self-decided

  Scenario: an unknown request is refused
    When I send a POST to "/external/v1/access-requests/00000000-0000-0000-0000-000000000000/reject"
    Then the response status is 404
