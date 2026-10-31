Feature: GET /external/v1/engine/{driver}/connections/{name}/targets
  What a search platform indexes, so the console can offer a target.

  Background:
    Given the stack is running

  Scenario: elasticsearch lists the seeded index
    Given an elasticsearch connection named "search"
    And I seed elasticsearch document "1" in "uat_index" with {"name":"a"}
    Then the elasticsearch targets hold "uat_index"

  Scenario: meilisearch lists the seeded index
    Given a meilisearch connection named "meili"
    And I seed meilisearch index "uat_index" with {"id":"1","name":"a"}
    Then the meilisearch targets hold "uat_index"

  Scenario: the route needs a session
    Given no session
    When I send a GET to "/external/v1/engine/elasticsearch/connections/anything/targets"
    Then the response status is 401
