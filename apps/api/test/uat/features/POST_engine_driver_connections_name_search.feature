Feature: POST /external/v1/engine/{driver}/connections/{name}/search
  One query against one target; a delete is refused here and raised as a change.

  Background:
    Given the stack is running

  Scenario: elasticsearch answers a query
    Given an elasticsearch connection named "search"
    And I seed elasticsearch document "1" in "uat_index" with {"name":"a"}
    When I search elasticsearch index "uat_index" for {"match_all":{}}
    Then the search result has 1 document(s)

  Scenario: the route needs a session
    Given no session
    When I send a POST to "/external/v1/engine/elasticsearch/connections/anything/search" with body:
      """
      {"target":"uat_index","op":"search","query":"{\"match_all\":{}}"}
      """
    Then the response status is 401
