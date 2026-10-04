Feature: MongoDB exploration, requests and stats
  The /v1/mongodb namespace explores a live server: databases and collections
  with their sizes, CRUD requests with the filter and the value kept apart, and
  the metrics worth watching. Requests run directly, so the guards are the op
  allowlist and the refusal to update or delete without a filter.

  Scenario: databases and collections carry their sizes
    Given the stack is running
    And a mongo connection named "docs"
    And the mongo server answers
    And I seed mongo collection "items" with [{"sku": "a", "qty": 1}, {"sku": "b", "qty": 2}]
    When the mongo database "demo" exists
    Then the mongo database "demo" holds collection "items"

  Scenario: a find request renders documents
    Given the stack is running
    And a mongo connection named "docs"
    And I seed mongo collection "reads" with [{"sku": "a"}, {"sku": "b"}]
    When I run a mongo "find" request on "reads" with filter {"sku": "a"} and value {}
    Then the mongo result has 1 row(s)

  Scenario: insert, update and delete a document
    Given the stack is running
    And a mongo connection named "docs"
    When I run a mongo "insert" request on "writes" with filter {} and value {"sku": "new", "qty": 5}
    Then the mongo result reports 1 affected
    When I run a mongo "update" request on "writes" with filter {"sku": "new"} and value {"$set": {"qty": 9}}
    Then the mongo result reports 1 affected
    When I run a mongo "delete" request on "writes" with filter {"sku": "new"} and value {}
    Then the mongo result reports 1 affected

  Scenario: a delete without a filter is refused
    Given the stack is running
    And a mongo connection named "docs"
    When I run a mongo "delete" request on "writes" with filter {} and value {}
    Then the mongo request is rejected with "needs a filter"

  Scenario: stats describe the server and the collection
    Given the stack is running
    And a mongo connection named "docs"
    And I seed mongo collection "stats" with [{"sku": "a"}, {"sku": "b"}]
    When I read mongo stats for "demo" and "stats"
    Then the mongo stats describe the server and the collection
