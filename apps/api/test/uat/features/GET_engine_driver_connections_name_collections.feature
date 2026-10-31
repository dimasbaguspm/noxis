Feature: GET /external/v1/engine/{driver}/connections/{name}/collections
  The collections one database holds; CouchDB answers its single keyspace in the same slot.

  Background:
    Given the stack is running

  Scenario: mongo lists a collection of its database
    Given a mongo connection named "mongo"
    And the mongo server answers
    And I seed mongo collection "customers" with [{"name":"a"}]
    Then the mongo database "demo" holds collection "customers"

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/mongo/connections/anything/collections?database=demo"
    Then the response status is 401
