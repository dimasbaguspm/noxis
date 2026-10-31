Feature: POST /external/v1/engine/{driver}/changes
  A change is proven before it is recorded: its rollback runs against it inside a transaction that always rolls back.

  Background:
    Given the stack is running
    And a clean registry

  Scenario Outline: <driver> validates a change against its own engine
    Given connection "submit_<driver>" is registered for driver "<driver>"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_submit_<driver> (id INT)" and rollback "DROP TABLE uat_submit_<driver>"
    Then the change is "validated"

    Examples:
      | driver    |
      | postgres  |
      | mysql     |
      | mariadb   |
      | cockroach |
      | sqlite    |

  # MySQL and MariaDB commit DDL implicitly, so they prove a change in a throwaway database; the others roll the
  # transaction back and report no scratch schema at all.
  Scenario Outline: <driver> proves the change in a scratch schema
    Given connection "submit_scratch_<driver>" is registered for driver "<driver>"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_scratch_<driver> (id INT)" and rollback "DROP TABLE uat_scratch_<driver>"
    Then the change is "validated"
    And the change was verified in a scratch schema

    Examples:
      | driver  |
      | mysql   |
      | mariadb |

  Scenario Outline: <driver> refuses a rollback that does not undo the change
    Given connection "submit_bad_<driver>" is registered for driver "<driver>"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_bad_<driver> (id INT)" and rollback "SELECT 1"
    Then the change is "failed"

    Examples:
      | driver    |
      | postgres  |
      | mysql     |
      | mariadb   |
      | cockroach |
      | sqlite    |

  Scenario Outline: <driver> has no change queue
    Then a POST to "/changes" for driver "<driver>" is refused naming the driver

    Examples:
      | driver    |
      | redis     |
      | valkey    |
      | dragonfly |

  Scenario: a change without a rollback is refused
    Given connection "submit_norollback" is registered for driver "postgres"
    When I send a POST to "/external/v1/engine/postgres/changes" with body:
      """
      {"connection":"submit_norollback","query":"CREATE TABLE uat_no_rollback (id INT)"}
      """
    Then the response status is 400
