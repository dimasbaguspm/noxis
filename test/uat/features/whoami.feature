Feature: Caller identity
  The API answers a caller its own name and scopes, and refuses a request that carries
  no token at all. Identity is what the journal records, so it has to be the token's
  name and not the permission it happens to hold.

  Scenario: the process answers anyone who asks whether it is up
    When I ask for the health of the server without a token
    Then the request succeeds
    And the reported version names the build

  Scenario: a token holder learns its own identity
    Given I hold the agent token
    When I ask who I am
    Then the request succeeds
    And my name is "agent"
    And my scopes are "read, propose"

  Scenario: the reviewer's token is a different identity with more scopes
    Given I hold the ops token
    When I ask who I am
    Then the request succeeds
    And my name is "ops"
    And my scopes are "read, propose, approve, apply, admin"

  Scenario: a request without a token is refused
    When I ask who I am without a token
    Then the request fails with status 401
