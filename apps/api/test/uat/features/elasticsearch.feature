Feature: Elasticsearch indices and a reviewed delete
  A search platform is neither SQL nor a document store: the console browses what
  it indexes, runs the platform's own query document against one index, and a
  delete is reviewed with the impact the cluster itself reports.

  Background:
    Given the stack is running

  Scenario: browse and query an index
    Given an elasticsearch connection named "es"
    And I seed elasticsearch document "uat_ada" in "demo" with {"user":"uat-ada","role":"admin"}
    When the elasticsearch targets hold "demo"
    And I search elasticsearch index "demo" for {"term":{"user.keyword":"uat-ada"}}
    Then the search result has 1 document(s)

  Scenario: a delete by query reports its impact before approval
    Given an elasticsearch connection named "es"
    And I seed elasticsearch document "uat_doomed" in "demo" with {"user":"uat-doomed"}
    When I request an elasticsearch "delete" on "demo" with query {"term":{"user.keyword":"uat-doomed"}}
    Then the search request reports an impact of 1
