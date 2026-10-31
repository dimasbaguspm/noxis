Feature: a connection's schema can be drawn as an ER diagram
  As an operator I want to see how a database fits together from the keys it
  actually declares, so the diagram never invents a relationship.

  Background:
    Given the stack is running

  Scenario Outline: declared keys and foreign keys reach the diagram on <driver>
    Given connection "erd_<driver>" is registered for driver "<driver>"
    When I submit a change with query "CREATE TABLE IF NOT EXISTS uat_erd_parent (id INT PRIMARY KEY)" and rollback "DROP TABLE uat_erd_parent"
    And I approve the change
    And I apply the change
    And I submit a change with query "CREATE TABLE IF NOT EXISTS uat_erd_child (id INT PRIMARY KEY, parent_id INT, code VARCHAR(16) UNIQUE, FOREIGN KEY (parent_id) REFERENCES uat_erd_parent (id))" and rollback "DROP TABLE uat_erd_child"
    And I approve the change
    And I apply the change
    And I submit a change with query "INSERT INTO uat_erd_parent (id) VALUES (1)" and rollback "DELETE FROM uat_erd_parent WHERE id = 1"
    And I approve the change
    And I apply the change
    And I submit a change with query "INSERT INTO uat_erd_child (id, parent_id, code) VALUES (1, 1, 'a')" and rollback "DELETE FROM uat_erd_child WHERE id = 1"
    And I approve the change
    And I apply the change
    Then the ERD of "erd_<driver>" marks "id" as a primary key
    And the ERD of "erd_<driver>" links "uat_erd_child" to "uat_erd_parent"
    And the ERD of "erd_<driver>" reports an index over "code" on "uat_erd_child"
    And the ERD of "erd_<driver>" reports a size for "uat_erd_child"

    Examples:
      | driver   |
      | postgres |
      | mysql    |
      | mariadb  |
      | cockroach |
