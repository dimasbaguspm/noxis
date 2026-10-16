# [security]: encrypt connection passwords at rest

- **Plan id:** issue 43
- **Area:** API & auth
- **Status:** completed 2026-10-09
- **Issue:** [#43](https://github.com/dimasbaguspm/noxis/issues/43)
- **Pull request:** [#47](https://github.com/dimasbaguspm/noxis/pull/47)
- **Commit:** [468abe2](https://github.com/dimasbaguspm/noxis/commit/468abe26e31a0c2df998c139c1d48886fe81d81d) — `[ai:y][#43] feat: encrypt connection passwords at rest (#47)`

## Plan

## What's problem?

Connection passwords are stored as plaintext in the database file, so anyone who reads that file has every credential. They cannot be hashed: noxis must send them to the database to authenticate.

## What's affected and how?

- A new secret key environment variable, read with the rest of the config.
- Passwords are encrypted with AES-256-GCM on save and decrypted on read.
- Existing plaintext rows are migrated when the store opens.
- A changed or missing key fails clearly instead of returning garbage.

## Definition of Done

- A new database stores ciphertext only; the file is useless without the key.
- Existing instances keep working after migration.
- A missing or wrong key produces an explicit startup or read error.
- API responses never include the password.

## Additional Notes

- Decide before implementing: should the key be mandatory, or an optional plaintext fallback with a loud warning?

---
_Auto-snapshotted from GitHub so the plan survives outside the tracker. Do not edit by hand; regenerate from the issue/PR._
