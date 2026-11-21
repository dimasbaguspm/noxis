---
name: noxis-setup
description: Install and register a MySQL or MariaDB instance in noXis. Use on first contact with a noXis deployment, when the operator asks to start governing a database, or when a connection has to be corrected.
---

# noXis setup

Registering a database is a decision about **who may change it**. Do not make that decision for the
operator: ask, then write exactly what they approved.

## 1. Ask before you write anything

Ask these four questions and wait for the answers:

1. **Which server, and which schema?** Host, port, engine (`mysql` or `mariadb`), and the one schema this
   target governs. MySQL and MariaDB call a schema a *database*; a server with several becomes several
   targets, which is deliberate — the review boundary is per schema.
2. **What may I do, and what stays yours?** The five capabilities are separable:

   | Scope | Grants |
   | --- | --- |
   | `read` | query rows, list instances/schemas/tables, read proposals and their events |
   | `propose` | create proposals and re-run rehearsals — nothing is applied |
   | `approve` | approve or reject a rehearsed proposal |
   | `apply` | run an approved proposal against the target |
   | `admin` | register, edit and archive instances |

   An agent that reads and proposes never needs `approve` or `apply`.
3. **Who decides?** Approving and applying should be a different token holder. Ask who.
4. **May I register and archive instances?** That is `admin` and is not needed to change a schema.

## 2. Write one token per caller

`TOKENS` is inline JSON on the server; the **name** is what the journal records, so name it after the
caller (a person, an agent, a CI job) and never after the permission.

```bash
TOKENS='{
  "agent-claude": {"secret": "<32+ chars>", "scopes": ["read", "propose"]},
  "ops-dimas":    {"secret": "<32+ chars>", "scopes": ["read", "propose", "approve", "apply", "admin"]}
}'
```

`SECRET_KEY` (32 bytes hex/base64) seals target passwords; without it a target with a password cannot be
created. A row sealed with another key refuses to open and says to re-enter the password.

## 3. Prove the connection before it becomes a row

```bash
noxis target test --host db.internal --port 3306 --driver mysql \
  --username noxis --password '…' --tls-mode skip-verify
```

The answer includes the **schemas those credentials can see**. TLS: `preferred` (default), `verify`,
`skip-verify` (self-signed — development only).

## 4. Register

```bash
noxis target add --name prod --host db.internal --port 3306 --driver mysql \
  --database app --username noxis --password '…' --tls-mode verify \
  --idempotency-key register-prod
```

`--database` is the schema the target governs, and it is required: it is the connection's default schema
and the schema the rehearsal rebuilds. A moved server is a **new** target, not an edit.

## 5. Correcting a connection

```bash
noxis target set <ref> --password '…' --probe     # --probe requires --password: the stored one is never read back
noxis target set <ref> --username noxis --name prod-2 --tls-mode verify
```

Fields an edit may change: name, username, password, TLS mode, options. Host, port and schema are the
instance's identity.

## 6. Archive, never delete

```bash
noxis target archive <ref>
```

It leaves the list; its proposals and journal stay. There is no un-archive.

## 7. Grants on the governed server (optional)

A rehearsal builds a throwaway database and drops it again:

```sql
GRANT CREATE, DROP ON *.* TO 'noxis'@'%';
GRANT ALL PRIVILEGES ON `noxis_rehearse_%`.* TO 'noxis'@'%';
```

Without them nothing breaks — the rehearsal degrades to **static** (parse + plan + row estimate) and says
so by name.

## Never

- Never invent a scope for yourself, or reuse the operator's token because it "already works".
- Never put a password in a command that lands in shell history when the API or a secret store is available.
- Never archive a target whose proposals are still open without saying so out loud.
