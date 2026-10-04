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

  Scenario: an insert is requested, reviewed and applied
    Given the stack is running
    And a mongo connection named "docs"
    When I request a mongo "insert" on "queued" with filter {} and value {"sku": "z", "qty": 3}
    Then the mongo request is validated
    And the mongo request estimates 1 affected
    When I approve the mongo request
    And I apply the mongo request
    Then the mongo request is "applied"
    When I run a mongo "find" request on "queued" with filter {"sku": "z"} and value {}
    Then the mongo result has 1 row(s)

  Scenario: a delete without a filter is refused by the queue
    Given the stack is running
    And a mongo connection named "docs"
    And I seed mongo collection "review" with [{"sku": "d"}]
    When I request a mongo "delete" on "review" with filter {} and value {}
    Then the mongo request is "failed"
    And the mongo request checks mention "needs a filter"
    And the mongo request was not recorded

  Scenario: a write is requested, reviewed and applied
    Given the stack is running
    And a mongo connection named "docs"
    And I seed mongo collection "review" with [{"sku": "a"}, {"sku": "b"}]
    When I request a mongo "update" on "review" with filter {"sku": "a"} and value {"$set": {"qty": 7}}
    Then the mongo request is validated
    And the mongo request estimates 1 affected
    When I approve the mongo request
    Then the mongo request is "approved"
    When I apply the mongo request
    Then the mongo request is "applied"

  Scenario: a request that fails its dry run is not recorded
    Given the stack is running
    And a mongo connection named "docs"
    And I seed mongo collection "review" with [{"sku": "c"}]
    When I request a mongo "update" on "review" with filter {"sku": "c"} and value {"$sett": {"qty": 1}}
    Then the mongo request is "failed"
    And the mongo request was not recorded

  Scenario: a read cannot be queued as a request
    Given the stack is running
    And a mongo connection named "docs"
    When I request a mongo "find" on "review" with filter {} and value {}
    Then the mongo request is "failed"
    And a read op is refused by the request queue

  Scenario: a write is refused by the read endpoint
    Given the stack is running
    And a mongo connection named "docs"
    When I run a mongo "delete" request on "review" with filter {"sku": "a"} and value {}
    Then the mongo request is rejected with "for review"

  Scenario: stats describe the server and the collection
    Given the stack is running
    And a mongo connection named "docs"
    And I seed mongo collection "stats" with [{"sku": "a"}, {"sku": "b"}]
    When I read mongo stats for "demo" and "stats"
    Then the mongo stats describe the server and the collection
