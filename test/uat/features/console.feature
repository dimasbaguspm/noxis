Feature: operator console
  As an operator I want a browser console served by the master itself, so
  the control plane is usable without curl.

  Scenario: the root path serves the console page
    Given the stack is running
    Then the operator console is served at "/"