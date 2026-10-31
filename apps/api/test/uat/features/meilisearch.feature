Feature: Meilisearch indexes and a reviewed delete
  Meilisearch answers its own query parameters and applies a write as a task, so
  the console searches an index and a reviewed delete waits for the task the
  server accepted instead of recording a queue id as an outcome.

  Background:
    Given the stack is running

  Scenario: browse and query an index
    Given a meilisearch connection named "meili"
    And I seed meilisearch index "demo" with [{"id":"uat_ada","user":"uat-ada","role":"admin"}]
    When the meilisearch targets hold "demo"
    And I search meilisearch index "demo" for {"filter":"user = 'uat-ada'"}
    Then the meilisearch result has 1 document(s)

  Scenario: a delete by filter reports its impact before approval
    Given a meilisearch connection named "meili"
    And I seed meilisearch index "demo" with [{"id":"uat_doomed","user":"uat-doomed","role":"viewer"}]
    When I request a meilisearch "delete" on "demo" with query {"filter":"user = 'uat-doomed'"}
    Then the meilisearch request reports an impact of 1
