@access
Feature: grants, access requests and the account directory
  An operator grants a level on one connection, a user asks for access, and an
  admin decides; nobody decides their own request, and the account directory
  stays readable and free of secrets.

  Background:
    Given the stack is running
    And connection "demo" is registered

  Scenario: a grant is set, listed and revoked
    When I grant "local:uat-user@example.com" the "read" level on the current connection
    Then the permission list shows "local:uat-user@example.com" at "read"
    When I revoke "local:uat-user@example.com" from the current connection
    Then the permission list is empty

  Scenario: a request is filed and cannot be self-approved
    Given a fresh connection "demo_req" is registered
    When I request the "read" level on the current connection
    Then the access request list contains it
    When I approve the access request
    Then the request is refused as self-decided

  Scenario: a request is filed and cannot be self-rejected
    Given a fresh connection "demo_req2" is registered
    When I request the "write" level on the current connection
    Then the access request list contains it
    When I reject the access request
    Then the request is refused as self-decided

  Scenario: the access matrix lists the connection
    Then the access matrix lists the current connection

  Scenario: the account directory is readable without secrets
    Then the user directory contains the root account
    And the root profile carries its email

  Scenario: an operator cannot change their own account
    Then changing my own account is refused
