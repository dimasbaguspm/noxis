Feature: one SQL engine that behaves the same on every supported database
  As an operator I want the console's SQL operations to work the same on
  postgres, mysql and mariadb, so a connection's driver only changes how noxis
  talks to it, not what an operator can do.

  Background:
    Given the stack is running

  Scenario Outline: a change validates, applies and is browsable on <driver>
    Given connection "compat_<driver>" is registered for driver "<driver>"
    And the connection test passes
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_compat (id INT, note VARCHAR(64))" and rollback "DROP TABLE uat_compat"
    Then the change is "validated"
    And I approve the change
    And I apply the change
    Then the change is "applied"
    And the schema lists table "uat_compat"
    When I read "SELECT COUNT(*) AS n FROM uat_compat"
    Then the result has 1 row(s)
    And the first row's "n" is "0"

    Examples:
      | driver   |
      | postgres |
      | mysql    |
      | mariadb  |

  Scenario Outline: analyze and a row preview work on <driver>
    Given connection "compat_<driver>" is registered for driver "<driver>"
    And the connection test passes
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_compat_preview (id INT, note VARCHAR(64))" and rollback "DROP TABLE uat_compat_preview"
    And I approve the change
    And I apply the change
    Then analyze on table "uat_compat_preview" succeeds
    And table "uat_compat_preview" has 0 row(s)

    Examples:
      | driver   |
      | postgres |
      | mysql    |
      | mariadb  |

  Scenario Outline: a DML dry run rolls back on <driver>
    Given connection "compat_<driver>" is registered for driver "<driver>"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_compat_dry (id INT)" and rollback "DROP TABLE uat_compat_dry"
    And I approve the change
    And I apply the change
    Then I dry-run "INSERT INTO uat_compat_dry (id) VALUES (4242)"
    And I read "SELECT COUNT(*) AS n FROM uat_compat_dry WHERE id = 4242"
    Then the result has 1 row(s)
    And the first row's "n" is "0"

    Examples:
      | driver   |
      | postgres |
      | mysql    |
      | mariadb  |

  Scenario Outline: a DDL dry run is refused where the engine commits DDL implicitly
    Given connection "compat_<driver>" is registered for driver "<driver>"
    Then I dry-run "CREATE TABLE uat_compat_refused (id INT)" and it is refused

    Examples:
      | driver  |
      | mysql   |
      | mariadb |
