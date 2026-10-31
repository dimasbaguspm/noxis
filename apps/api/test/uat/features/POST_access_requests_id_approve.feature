Feature: POST /external/v1/access-requests/{id}/approve
  Nobody decides their own request, so an approval needs a second person.

  Background:
    Given the stack is running
    And a clean registry
    And connection "approve_pg" is registered for driver "postgres"

  Scenario: the requester cannot approve their own request
    When I request the "read" level on the current connection
    Then the access request list contains it
    When I approve the access request
    Then the request is refused as self-decided

  Scenario: an unknown request is refused
    When I send a POST to "/external/v1/access-requests/00000000-0000-0000-0000-000000000000/approve"
    Then the response status is 404
