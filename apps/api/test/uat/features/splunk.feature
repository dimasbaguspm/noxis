@slow @splunk
Feature: Splunk events and a reviewed delete
  Splunk is a search and observability deployment: its query language is SPL and
  its results are events. Its image is large and licence-gated, so this suite is
  opt-in: `docker compose --profile splunk up -d splunk`, then
  `UAT_KEEP=1 UAT_TAGS=@splunk make uat`.

  Background:
    Given the stack is running

  Scenario: run an SPL search
    Given a splunk connection named "spl"
    When I search splunk for "search index=_internal | head 3"
    Then the splunk result has 3 event(s)

  Scenario: a delete reports its impact before approval
    Given a splunk connection named "spl"
    When I request a splunk "delete" with search "search index=_internal sourcetype=splunkd | head 1"
    Then the splunk request reports an impact of 1
