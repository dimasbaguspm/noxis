Feature: CouchDB documents and a reviewed write
  CouchDB is a document store with a JSON API: the console browses its databases
  and documents, and a write is proved in a throwaway database before it is
  applied, because a CouchDB write has no rollback.

  Background:
    Given the stack is running

  Scenario: browse a database of documents
    Given a couchdb connection named "couch"
    And the couchdb database "demo" exists
    And I seed couchdb document "uat_ada" with {"user":"uat-ada","role":"admin"}
    When the couchdb database list holds "demo"
    And I run a couchdb find on "demo" with selector {"user":"uat-ada"}
    Then the couchdb result has 1 document(s)

  Scenario: a document write is proved in a scratch database
    Given a couchdb connection named "couch"
    And the couchdb database "demo" exists
    When I request a couchdb "insert" on "demo" with value {"_id":"uat-couch-write","user":"uat-grace"}
    Then the couchdb request is validated in a scratch database
