Feature: POST /external/v1/engine/{driver}/connections/{name}/requests
  A document read runs directly; a write is refused here and raised as a change.

  Background:
    Given the stack is running

  Scenario: a mongo read returns its grid
    Given a mongo connection named "mongo"
    And the mongo server answers
    And I seed mongo collection "customers" with [{"name":"a"}]
    When I run a mongo "find" request on "demo.customers" with filter {} and value {}
    Then the response status is 200

  Scenario: a mongo write is refused by the read path
    Given a mongo connection named "mongo"
    And the mongo server answers
    When I request a mongo "update" on "demo.customers" with filter {"id":1} and value {"$set":{"name":"x"}}
    Then a read op is refused by the request queue

  Scenario: the route needs a session
    Given no session
    When I send a POST to "/external/v1/engine/mongo/connections/anything/requests" with body:
      """
      {"database":"demo","collection":"customers","op":"find"}
      """
    Then the response status is 401
