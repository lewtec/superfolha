# Superfolha Product & Architecture Spec

| Field | Value |
|-------|-------|
| **Title** | Superfolha Product & Architecture Spec |
| **Status** | Draft (ephemeral forge, signer-socket grill 2026-08-24) |
| **Date** | 2026-08-24 |
| **Repo** | `/home/lucasew/WORKSPACE/OPENSOURCE-own/superfolha` |
| **Audience** | Engineers implementing Superfolha |

This document is the product and architecture specification. It replaces the July 2026 collab spec where they conflict. Live CRDT rules that do not mention accounts or on-disk projects still apply.

---

## Terminology

| Term | Meaning |
|------|---------|
| **session** | One live editor for one `remote + branch`. Identity is a UUIDv7. |
| **host** | First GitHub user who created this incarnation by cloning. The host seat does not transfer while the process holds the session. |
| **remote** | Canonical identity of a git repo (HTTPS form, no `.git`). Transport is always SSH. |
| **session key** | Ed25519 seed in the browser, keyed by remote+branch. Used only for git SSH. Not the login key. |
| **signer tab** | The browser tab that started clone or Persist. Only that tab may `sign()`. |
| **preauth** | Signed token the host mints so another signed-in user can join this session. |
| **knock** | Join request that uses only the session UUIDv7. The host can disable knocks (auto-reject). |
| **hub** | In-process actor: ygo CRDT, clients, flush, commit+push lock. |

Do not use “room”, “project row”, or “owner” for these ideas. GitHub repo admin is not the Superfolha host.

---

## Overview

Superfolha is a web LaTeX editor. Durable paper state is the git remote. The server process is disposable.

A user signs in with an Ed25519 key in the browser. Starting work is an SSH clone. The browser holds the session key and signs git. Superfolha opens TCP to the forge and asks that tab to `sign()`. Collaborators join with a preauth token or, if the host allows it, a knock on the session id.

If the process dies, RAM dies. The next writer clones again from the remote. Unpushed work is gone.

---

## Goals

1. Ed25519 challenge-sign is identity. The login private key never leaves the browser.
2. SSH clone creates a session (UUIDv7). The form does not clone.
3. At most one live session per `remote + branch`. A second create fails. It does not join.
4. Session lives until the host ends it. No idle-evict.
5. The session private key never enters hub RAM. The server asks a signer tab to `sign()`.
6. Persist is started by a seed-holding tab. That same tab signs. Guests cannot persist.
7. Preauth and optional knock are the join doors. Clone is create-only.
8. Live multi-client CRDT editing stays (ygo + Yjs, one hub per session).
9. Single-instance process.

## Non-goals (this cut)

- Password or email accounts.
- Superfolha-owned project rows as source of truth.
- User login Ed25519 reused as a git deploy key.
- HTTP clone or a pasted token.
- Per-invite revoke lists and invite-epoch files (kick-everyone only).
- Multi-instance hubs.
- Durable CRDT snapshots.

---

## Decisions from grill (normative)

| # | Decision |
|---|----------|
| F1 | **Forge-native.** The git remote is the paper. Superfolha does not own a second copy of truth. |
| F2 | **SSH clone is the primitive.** The form is SSH only. No token. No `https://` clone. |
| F3 | **Ed25519 challenge-sign identity.** The browser holds the login private key. Login is nonce + signature. Fingerprint is the display name. |
| F4 | **Session key stays in the browser.** IndexedDB, per remote+branch. Load/store is JSON. Create posts only the public line. The server never receives the seed. Superfolha opens SSH and asks the signer tab to `sign()`. |
| F5 | **Signer tab.** Only the tab that started clone or Persist may sign. A second open tab does not sign for it. Close that tab and that action stops. |
| F6 | **Clone creates a session** with a UUIDv7. Disk path `{state-dir}/repos/{sessionID}`. |
| F7 | **Uniqueness:** at most one live session per `remote + branch`. Second clone **fails**. Do not put the UUIDv7 in that error. Host retry (same GitHub login) returns the existing session. |
| F8 | **No join via clone.** Join is preauth or knock. |
| F9 | **Host** = first writer who created this incarnation. No live transfer. After process death, the next create makes a new UUIDv7 and a new host. |
| F10 | **Session stays until the host ends it.** Process nuke still drops it. |
| F11 | **Knock is opt-in.** Auto-reject closes only the knock door. |
| F12 | **Walk-in:** host, users already admitted (preauth redeem), and (if knock is on) users the host admits. GitHub write on the remote does **not** walk in through the clone form. |
| F13 | **v1 revoke:** kick everyone (drop live clients). Preauth TTL/epoch later. |
| F14 | **Persist:** a seed-holding tab owns a 30s dirty cooldown and the Persist button. That tab asks and signs. Guest Persist is an error: no commit, no push. Guest keystrokes dirty the host tab’s doc; the host tab’s cooldown may persist. Server singleflight only. No server auto-push. |
| F15 | **GitHub App leftover is unused.** No OAuth identity. No installation token for clone. |
| F16 | Login key and session key are different. One session pubkey per remote+branch (GitHub deploy keys cannot be reused across repos). |
| F17 | **Create stays pending.** Sessions page opens a signer socket. Retry clone uses that socket. After clone, the same tab signs a probe push of current HEAD. Fail → key modal, stay on sessions. Success → editor. |
| F18 | If `main.tex` is missing after clone, write the default template locally. It is committed on the first Persist. |

CRDT decisions that still hold: one Y.Doc per session; CRDT is RAM-only; Git working tree is the disk projection; stream access = session access; flush before compile; lock CRDT sync while commit+push runs; Synced = server ack, not push; artifacts stay outside git; single instance.

---

## Identity

1. User opens `/login`. The browser creates or reuses an Ed25519 key in IndexedDB.
2. `GET /login/challenge` returns a short-lived signed nonce. The browser signs it. `POST /login/verify` checks the signature.
3. Superfolha sets a JWT cookie: `{key_fingerprint_id, login}`. No user row. The identity private key never leaves the browser.
4. The browser holds a **session** Ed25519 seed (IndexedDB, per remote+branch). Create posts only the public line. The host adds that public key on the remote. Clone and push ask that tab to `sign()`. The login key is never uploaded.

Public remotes still require this login.

---

## Start a session

1. Signed-in user pastes an SSH remote and a branch (default `main`). The browser loads or mints a session seed for that pair and shows the public key. Load/store export that seed as JSON.
2. Submit posts `{remote, branch, ssh_public}`. No seed.
3. Server canonicalizes the remote. If a live session exists for that pair:
   - same host login → return that session
   - anyone else → fail, no id
4. Server stores a **pending** session. It does not clone. Redirect stays on `/sessions`. The host adds the public key on the remote.
5. The sessions page opens `GET /ws/sessions/{id}`. That tab is the signer. The server clones over SSH, asking the tab to `sign()`.
6. After clone, if `main.tex` is missing, write the default template. Then the same tab signs a probe push of current HEAD.
7. Probe fail → error + key modal. Stay pending. Probe success → open the hub. Redirect to `/editor/{sessionID}`.

---

## Join

| Door | Who | Effect |
|------|-----|--------|
| Host | Creator of this incarnation | Always in |
| Preauth | Any signed-in user with a valid token | Admit. Skip knock |
| Knock | Signed-in user who has the UUIDv7 | Queued only if knock is enabled. Host admits or ignores |
| Clone | — | Never a join door |

Preauth is a signed `{session_id, exp}` using the deploy secret. v1 does not persist a revoke list. Host “kick everyone” disconnects clients. They can redeem a still-valid preauth again until the host ends the session.

Do not treat the UUIDv7 as a secret capability by itself unless knock is on.

---

## Persist

```text
seed-holding tab: Persist click or 30s dirty cooldown
    → that tab asks (commit.now)
    → singleflight
    → flush CRDT → git commit (author = host login) → git push
         (each SSH sign() is a WS round-trip to that same tab)
    → unlock
```

Dirty means the Yjs doc, including peer edits. Only a tab that holds the seed runs the cooldown.

Guest Persist → error. No commit. No push. The server does not ask another tab to sign.

If push fails, keep the local commit, report the error, unlock. No second commit+push while one flight is running.

The server does not auto-commit. Close the signer tab and push stops.

---

## Process death

Gone: hubs, CRDT, working trees, knock queues, in-memory admits.

Still valid: git remotes, JWT cookie (until expiry), session seed in the host browser, deploy secrets.

Next writer signs in, clones the same remote+branch, becomes host of a new UUIDv7.

---

## Live collab (unchanged shape)

One hub / one Y.Doc per session. Text in CRDT. Tree listing from the working tree. Blobs on disk only. Session fence uses the session UUIDv7 as `session_id` (hello / hello.ack). Compile is `GET /api/compile` after flush. Chat is RAM-only.

Hub idle-evict from the July spec is **withdrawn**. The host ends the session.

---

## HTTP routes

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/` | Landing |
| GET | `/login` | GitHub sign-in page |
| GET | `/login/github` | Start OAuth |
| GET | `/login/github/callback` | OAuth callback |
| POST | `/logout` | Clear cookie |
| GET | `/sessions` | Live sessions this user may open |
| POST | `/sessions` | Create pending session |
| GET | `/ws/sessions/{id}` | Host signer socket: clone + probe push |
| POST | `/sessions/{id}/end` | Host ends session |
| POST | `/sessions/{id}/preauth` | Host mints preauth |
| POST | `/sessions/{id}/knock` | Request admit |
| POST | `/sessions/{id}/admit` | Host admits a login |
| POST | `/sessions/{id}/kick` | Host drops live clients |
| POST | `/sessions/{id}/knock-mode` | Host sets knock on/off |
| GET | `/editor/{id}` | Editor. `?preauth=` redeems |
| GET | `/ws/projects/{id}` | Live WS (session id) |
| GET | `/api/compile` | Compile |

`/register` is gone. `/projects` redirects to `/sessions`.

### Signer wire

Tab → server: `clone` `{public}`, `hello.ack` `{session_id, ssh_public}`, `commit.now`, `ssh.sign.ok` `{id, signature}`, `ssh.sign.err` `{id, message}`.

Server → tab: `ssh.sign` `{id, data}`, `clone.ok`, `clone.err` `{message, ssh_public}`, `error` `{message}` (`persist.no_key` when the tab has no session key).

`data` and `signature` are standard base64. `signature` is the raw 64-byte Ed25519 signature.

---

## Configuration

| Env | Role |
|-----|------|
| `JWT_SECRET` | Cookie and preauth signatures (required in production) |
| `STATE_DIR` | Working trees + artifacts |
| `GO_ENV=development` | Allows JWT fallback |

SQLite/Postgres user and project tables are not used for identity or session lists. A driver may remain for a transition; it is not the product store.

---

## Module layout (added)

```text
internal/remote/      # canonicalize remotes; SSH-only validate
internal/session/     # hub + live registry (uniqueness, admit, knock, signer clone)
internal/git/         # clone, commit, push; SessionSSH (tab signer or test key)
```

---

## Risks

| Risk | Mitigation |
|------|------------|
| Unpushed work lost on nuke | Signer-tab cooldown + Persist; UI says Synced ≠ pushed |
| Guessable session URL | Knock off by default. Fail-on-duplicate does not leak the id |
| Read-only deploy key | Probe push after clone, before the editor |
| Host tab closed | No `sign()`. Persist and clone wait. |
| Two writers race create | Registry lock on `remote + branch` |

---

## Test strategy

| Layer | What |
|-------|------|
| remote | Canonical URL; GitHub owner/repo parse |
| session registry | Duplicate create fails; host retry returns same id; preauth admit; knock off rejects |
| git | Clone from a local remote; commit+push singleflight |
| githubapp | httptest for OAuth exchange and installation token |
| HTTP | Login page; SSH form creates pending session; signer WS clones; second user cannot clone-join |

---

## Grill log (2026-08-24)

Forge-native → Ed25519 login, session SSH key in the browser → SSH only → create pending, sessions signer socket clones then probe-pushes → one live session per remote+branch → host is first writer of this incarnation → knock opt-in → persist = signer tab (click or 30s dirty cooldown), guest Persist errors → seed never POSTed, load/store JSON → main.tex seeded locally if missing.
