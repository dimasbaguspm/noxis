# Plan archive

Every initiative planned and executed with the `ai:y` tag, snapshotted from the
GitHub tracker into this directory by plan id. One file per plan; the README is
the index.

**Convention:** a plan is one issue (or PR when no issue exists). The issue body
is the plan of record and follows the sections
_What's problem? → What's affected and how? → Definition of Done → Additional
Notes_. Each commit carries `[ai:y][#<issue>] <type>(<scope>): <subject> (#<pr>)`.
This directory makes the plan survive even if the tracker does not.

**Fresh session:** read the most recent plans under `backend/`-tagged areas to
learn the current architecture, then `AGENTS.md` + `docs/agent/`.

## Index

### API & auth (10)

- [[api]: serve every engine family through one registry-gated /v1/{engine} route table](001-serve-every-engine-family-through-one-registry-gated-v1engin.md) — plan issue #1, completed 2026-10-07
- [[api]: verify DDL and DML in a scratch schema](009-verify-ddl-and-dml-in-a-scratch-schema.md) — plan issue #9, completed 2026-10-08
- [[api]: schedule changes behind an internal tick](011-schedule-changes-behind-an-internal-tick.md) — plan issue #11, completed 2026-10-08
- [[api]: retire SSE and instance metrics endpoints](019-retire-sse-and-instance-metrics-endpoints.md) — plan issue #19, completed 2026-10-08
- [[api]: rename the SQL analytics op to analyze](021-rename-the-sql-analytics-op-to-analyze.md) — plan issue #21, completed 2026-10-08
- [[api]: authenticate callers with OIDC bearer tokens](022-authenticate-callers-with-oidc-bearer-tokens.md) — plan issue #22, completed 2026-10-08
- [[api]: store per-connection role grants](023-store-per-connection-role-grants.md) — plan issue #23, completed 2026-10-08
- [[api]: enforce per-connection roles on connection routes](024-enforce-per-connection-roles-on-connection-routes.md) — plan issue #24, completed 2026-10-08
- [[security]: lock instance identity and allow password rotation](042-lock-instance-identity-and-allow-password-rotation.md) — plan issue #42, completed 2026-10-09
- [[security]: encrypt connection passwords at rest](043-encrypt-connection-passwords-at-rest.md) — plan issue #43, completed 2026-10-09

### Backend (9)

- [[backend] C4 restructure: extract application layer, rename controller to api](166-c4-restructure-extract-application-layer-rename-controller-t.md) — plan issue #166, completed 2026-10-10
- [[backend] restructure: usecases + modules layers, in-memory cache repo](168-restructure-usecases-modules-layers-in-memory-cache-repo.md) — plan issue #168, completed 2026-10-10
- [[backend] restructure: handlers + middlewares packages, deps injected once via request ctx](169-restructure-handlers-middlewares-packages-deps-injected-once.md) — plan issue #169, completed 2026-10-10
- [[backend] scheduler endpoint rename, drop tick key, user-attributed logging + full-layer spans](173-scheduler-endpoint-rename-drop-tick-key-user-attributed-logg.md) — plan issue #173, completed 2026-10-10
- [[backend] common plumbing: config package, ticker host env, drop otel metrics, shared limits](175-common-plumbing-config-package-ticker-host-env-drop-otel-met.md) — plan issue #175, completed 2026-10-10
- [[backend] wire shapes + identity into entity, AuthInfo into config](176-wire-shapes-identity-into-entity-authinfo-into-config.md) — plan issue #176, completed 2026-10-10
- [[backend] bottom layer purity: entity + interfaces import nothing else](180-bottom-layer-purity-entity-interfaces-import-nothing-else.md) — plan issue #180, completed 2026-10-10
- [[backend] usecases restructure: one file = one use case, handler seams via interfaces](181-usecases-restructure-one-file-one-use-case-handler-seams-via.md) — plan issue #181, completed 2026-10-10
- [[backend] usecases stand alone: drop Base, explicit deps, no cross-use-case imports](185-usecases-stand-alone-drop-base-explicit-deps-no-cross-use-ca.md) — plan issue #185, completed 2026-10-10

### CI (1)

- [ci: run Go/UAT and web jobs only when their files change](005-ci-run-gouat-and-web-jobs-only-when-their-files-change.md) — plan issue #5, completed 2026-10-07

### Changes & query (11)

- [[changes]: streamline the DML/DDL create flow and queue](049-streamline-the-dmlddl-create-flow-and-queue.md) — plan issue #49, completed 2026-10-09
- [[changes]: schedule an approved change](050-schedule-an-approved-change.md) — plan issue #50, completed 2026-10-09
- [[changes]: filter the DML/DDL queue by instance](052-filter-the-dmlddl-queue-by-instance.md) — plan issue #52, completed 2026-10-09
- [[query]: fix the schema and result panel layout](054-fix-the-schema-and-result-panel-layout.md) — plan issue #54, completed 2026-10-09
- [[changes]: paginate the DML/DDL queues](062-paginate-the-dmlddl-queues.md) — plan issue #62, completed 2026-10-09
- [[changes]: turn the create flow into an inline form](064-turn-the-create-flow-into-an-inline-form.md) — plan issue #64, completed 2026-10-09
- [[query]: free the editor row and kill the stray scrollbars](065-free-the-editor-row-and-kill-the-stray-scrollbars.md) — plan issue #65, completed 2026-10-09
- [[changes]: sharpen the DML/DDL detail](067-sharpen-the-dmlddl-detail.md) — plan issue #67, completed 2026-10-09
- [[changes]: restore the create dialog and drop the batch flow](080-restore-the-create-dialog-and-drop-the-batch-flow.md) — plan issue #80, completed 2026-10-09
- [[query]: label the left-panel fields](081-label-the-left-panel-fields.md) — plan issue #81, completed 2026-10-09
- [[query]: run queries on Ctrl+Enter](082-run-queries-on-ctrlenter.md) — plan issue #82, completed 2026-10-09

### Console (web) (46)

- [[web]: prefix console routes with the engine family and drive them from one family registry](002-prefix-console-routes-with-the-engine-family-and-drive-them.md) — plan issue #2, completed 2026-10-07
- [[web]: improve SQL Query page layout and readability](007-improve-sql-query-page-layout-and-readability.md) — plan issue #7, completed 2026-10-07
- [[web]: attach SQL files and verify DML/DDL changes in a batch](010-attach-sql-files-and-verify-dmlddl-changes-in-a-batch.md) — plan issue #10, completed 2026-10-08
- [[web]: DML/DDL create wizard and scheduled queue](012-dmlddl-create-wizard-and-scheduled-queue.md) — plan issue #12, completed 2026-10-08
- [[web]: DML/DDL review titles, field names and rollback CTA](013-dmlddl-review-titles-field-names-and-rollback-cta.md) — plan issue #13, completed 2026-10-08
- [[web]: retire SSE consumers, analytics pages and dashboard reachability](020-retire-sse-consumers-analytics-pages-and-dashboard-reachabil.md) — plan issue #20, completed 2026-10-08
- [[web]: OIDC login and permission-gated console](025-oidc-login-and-permission-gated-console.md) — plan issue #25, completed 2026-10-08
- [[console]: filter the dashboard through the instances API](053-filter-the-dashboard-through-the-instances-api.md) — plan issue #53, completed 2026-10-09
- [[console]: add an Empty state for not-found pages](063-add-an-empty-state-for-not-found-pages.md) — plan issue #63, completed 2026-10-09
- [[console]: rename Dashboard to Instances](066-rename-dashboard-to-instances.md) — plan issue #66, completed 2026-10-09
- [[console]: migrate the console forms to react-hook-form](077-migrate-the-console-forms-to-react-hook-form.md) — plan issue #77, completed 2026-10-09
- [[console]: swap form dialogs to Sheets](079-swap-form-dialogs-to-sheets.md) — plan issue #79, completed 2026-10-09
- [[console]: fix the sheet structure: header, body, pinned footer](094-fix-the-sheet-structure-header-body-pinned-footer.md) — plan issue #94, completed 2026-10-09
- [[console]: extract auth into an AuthProvider](095-extract-auth-into-an-authprovider.md) — plan issue #95, completed 2026-10-09
- [[console]: introduce TanStack Query and api hooks for the core endpoints](096-introduce-tanstack-query-and-api-hooks-for-the-core-endpoint.md) — plan issue #96, completed 2026-10-09
- [[console]: migrate memcache and mongodb pages to the query hooks](097-migrate-memcache-and-mongodb-pages-to-the-query-hooks.md) — plan issue #97, completed 2026-10-10
- [[console]: sweep data-fetching useState and useEffect into reusable hooks](098-sweep-data-fetching-usestate-and-useeffect-into-reusable-hoo.md) — plan issue #98, completed 2026-10-10
- [[console]: make the field set the single form-field API](099-make-the-field-set-the-single-form-field-api.md) — plan issue #99, completed 2026-10-10
- [[console]: sweep UI primitives for consistency](100-sweep-ui-primitives-for-consistency.md) — plan issue #100, completed 2026-10-10
- [[console]: restructure the sidebar on the shadcn sidebar set](109-restructure-the-sidebar-on-the-shadcn-sidebar-set.md) — plan issue #109, completed 2026-10-10
- [[console]: add a login page for unauthenticated sessions](110-add-a-login-page-for-unauthenticated-sessions.md) — plan issue #110, completed 2026-10-10
- [[console]: standardize sheet forms with a body wrapper and form-id submit](111-standardize-sheet-forms-with-a-body-wrapper-and-form-id-subm.md) — plan issue #111, completed 2026-10-10
- [[console]: validate every form with zod and react-hook-form](112-validate-every-form-with-zod-and-react-hook-form.md) — plan issue #112, completed 2026-10-10
- [[ai:y] feat(web): port the sidebar to the dashboard-01 structure](118-featweb-port-the-sidebar-to-the-dashboard-01-structure.md) — plan pr #118, completed 2026-10-10
- [[console]: polish the sidebar shell: static title and instance submenu](119-polish-the-sidebar-shell-static-title-and-instance-submenu.md) — plan issue #119, completed 2026-10-10
- [[console]: route mongodb explore to /mongodb/explore](120-route-mongodb-explore-to-mongodbexplore.md) — plan issue #120, completed 2026-10-10
- [[console]: let the query editor fill its frame](121-let-the-query-editor-fill-its-frame.md) — plan issue #121, completed 2026-10-10
- [[console]: rework the DML/DDL queue filters and status styling](122-rework-the-dmlddl-queue-filters-and-status-styling.md) — plan issue #122, completed 2026-10-10
- [[console]: rebuild the connection wizard with radio-backed selects](123-rebuild-the-connection-wizard-with-radio-backed-selects.md) — plan issue #123, completed 2026-10-10
- [[console]: move password rotation into a dedicated sheet](124-move-password-rotation-into-a-dedicated-sheet.md) — plan issue #124, completed 2026-10-10
- [[console]: record the author on every write](125-record-the-author-on-every-write.md) — plan issue #125, completed 2026-10-10
- [[console]: add the RBAC settings page for per-instance grants](126-add-the-rbac-settings-page-for-per-instance-grants.md) — plan issue #126, completed 2026-10-10
- [[console]: add the profile page and sidebar user](127-add-the-profile-page-and-sidebar-user.md) — plan issue #127, completed 2026-10-10
- [[console][web] surface Settings and Profile in the sidebar](138-surface-settings-and-profile-in-the-sidebar.md) — plan issue #138, completed 2026-10-10
- [[console][web] wrap the sidebar nav in the dashboard-01 group pattern](139-wrap-the-sidebar-nav-in-the-dashboard-01-group-pattern.md) — plan issue #139, completed 2026-10-10
- [[console][web] brand the sidebar header like dashboard-01](140-brand-the-sidebar-header-like-dashboard-01.md) — plan issue #140, completed 2026-10-10
- [[console][web] make Instances a plain clickable item](141-make-instances-a-plain-clickable-item.md) — plan issue #141, completed 2026-10-10
- [[console][web] reset the connection sheet on close](142-reset-the-connection-sheet-on-close.md) — plan issue #142, completed 2026-10-10
- [[console][web] drop the redundant grid on connection step two](143-drop-the-redundant-grid-on-connection-step-two.md) — plan issue #143, completed 2026-10-10
- [[console][web] add a Next action to the connection kind step](144-add-a-next-action-to-the-connection-kind-step.md) — plan issue #144, completed 2026-10-10
- [[console][web] standardise the DML/DDL sheet width](145-standardise-the-dmlddl-sheet-width.md) — plan issue #145, completed 2026-10-10
- [[console][web] split the instance page into filter, CTA and results tiles](146-split-the-instance-page-into-filter-cta-and-results-tiles.md) — plan issue #146, completed 2026-10-10
- [[console][web] tile the DML/DDL search page like the instance page](147-tile-the-dmlddl-search-page-like-the-instance-page.md) — plan issue #147, completed 2026-10-10
- [[console][web] sidebar asChild menu items render as empty rows](159-sidebar-aschild-menu-items-render-as-empty-rows.md) — plan issue #159, completed 2026-10-10
- [[console][web] label the filter fields on Instances, DML and DDL](161-label-the-filter-fields-on-instances-dml-and-ddl.md) — plan issue #161, completed 2026-10-10
- [[console][web] flatten the sidebar user card (no switcher)](162-flatten-the-sidebar-user-card-no-switcher.md) — plan issue #162, completed 2026-10-10

### Instances (4)

- [[instances]: give connections a uuid and open a detail page](040-give-connections-a-uuid-and-open-a-detail-page.md) — plan issue #40, completed 2026-10-09
- [[instances]: search and filter the dashboard by name and driver](041-search-and-filter-the-dashboard-by-name-and-driver.md) — plan issue #41, completed 2026-10-09
- [[instances]: tidy the instance detail page](051-tidy-the-instance-detail-page.md) — plan issue #51, completed 2026-10-09
- [[instances]: paginate the instances list](068-paginate-the-instances-list.md) — plan issue #68, completed 2026-10-09

### Other (1)

- [[ai:y] chore: drop unused tabs and textarea components](093-chore-drop-unused-tabs-and-textarea-components.md) — plan pr #93, completed 2026-10-10

