Feature: GET /external/v1/engine/{driver}/connections/{name}/databases
  The databases a document server holds.

  Background:
    Given the stack is running

  Scenario: mongo lists the database a write created
    Given a mongo connection named "mongo"
    And the mongo server answers
    And I seed mongo collection "customers" with [{"name":"a"}]
    Then the mongo database "demo" exists

  Scenario: couchdb lists the database the seed created
    Given a couchdb connection named "couchdb"
    Then the couchdb database "uat_docs" exists

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/mongo/connections/anything/databases"
    Then the response status is 401
