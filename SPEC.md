# Superfolha Product & Architecture Spec

| Field | Value |
|-------|-------|
| **Title** | Superfolha Product & Architecture Spec |
| **Status** | Draft (ephemeral forge, post grill 2026-08-24) |
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
| **remote** | Canonical HTTP git URL (no `.git` suffix). |
| **preauth** | Signed token the host mints so another signed-in user can join this session. |
| **knock** | Join request that uses only the session UUIDv7. The host can disable knocks (auto-reject). |
| **installation token** | Short-lived GitHub App token. Held in hub RAM. Used for HTTP clone and push. |
| **hub** | In-process actor: ygo CRDT, clients, flush, commit+push lock. |

Do not use “room”, “project row”, or “owner” for these ideas. GitHub repo admin is not the Superfolha host.

---

## Overview

Superfolha is a web LaTeX editor. Durable paper state is the git remote. The server process is disposable.

A user signs in with GitHub. That login is identity. Starting work is an HTTP clone. GitHub App grant-on-use fills the clone credential for GitHub remotes. The clone creates a session. Collaborators join with a preauth token or, if the host allows it, a knock on the session id.

If the process dies, RAM dies. The next writer clones again from the remote. Unpushed work is gone.

---

## Goals

1. GitHub App OAuth is the only identity.
2. HTTP clone creates a session (UUIDv7).
3. At most one live session per `remote + branch`. A second create fails. It does not join.
4. Session lives until the host ends it. No idle-evict.
5. Clone credentials live in hub RAM. Never disk. Never SQLite.
6. Anyone in the session can ask to persist. The hub also auto-commits and pushes. Both use one singleflight.
7. Preauth and optional knock are the join doors. Clone is create-only.
8. Live multi-client CRDT editing stays (ygo + Yjs, one hub per session).
9. Single-instance process.

## Non-goals (this cut)

- Password or email accounts.
- Superfolha-owned project rows as source of truth.
- User Ed25519 / deploy-key identity.
- Per-invite revoke lists and invite-epoch files (kick-everyone only).
- Multi-instance hubs.
- Durable CRDT snapshots.

---

## Decisions from grill (normative)

| # | Decision |
|---|----------|
| F1 | **Forge-native.** The git remote is the paper. Superfolha does not own a second copy of truth. |
| F2 | **HTTP clone is the primitive.** GitHub login is identity plus sugar that fills the clone URL and credential. |
| F3 | **GitHub required as identity.** No anonymous host or guest. |
| F4 | **GitHub App, grant on use.** Fine-grained by the repos the user actually opens. Classic `repo` OAuth is out. |
| F5 | **Installation token in hub RAM** for GitHub HTTP clone and push. Paste-URL + pasted token for non-GitHub remotes. |
| F6 | **Clone creates a session** with a UUIDv7. Disk path `{state-dir}/repos/{sessionID}`. |
| F7 | **Uniqueness:** at most one live session per `remote + branch`. Second clone **fails**. Do not put the UUIDv7 in that error. Host retry (same GitHub login) returns the existing session. |
| F8 | **No join via clone.** Join is preauth or knock. |
| F9 | **Host** = first writer who created this incarnation. No live transfer. After process death, the next create makes a new UUIDv7 and a new host. |
| F10 | **Session stays until the host ends it.** Process nuke still drops it. |
| F11 | **Knock is opt-in.** Auto-reject closes only the knock door. |
| F12 | **Walk-in:** host, users already admitted (preauth redeem), and (if knock is on) users the host admits. GitHub write on the remote does **not** walk in through the clone form. |
| F13 | **v1 revoke:** kick everyone (drop live clients). Preauth TTL/epoch later. |
| F14 | **Persist:** anyone may ask. Quiet auto-commit+push as well. One singleflight (existing commit lock). Commit author = host GitHub login. Pusher = installation token or the RAM HTTP credential. |
| F15 | **GitHub only.** Delete password register/login. No dual auth. Local UUID projects are not a product path. |
| F16 | No user Ed25519 as login. GitHub deploy keys cannot reuse one pubkey across repos. |

CRDT decisions that still hold: one Y.Doc per session; CRDT is RAM-only; Git working tree is the disk projection; stream access = session access; flush before compile; lock CRDT sync while commit+push runs; Synced = server ack, not push; artifacts stay outside git; single instance.

---

## Identity

1. User opens `/login` and continues with GitHub (GitHub App user OAuth).
2. Callback issues a Superfolha JWT cookie: `{github_user_id, login}`. No user row.
3. JWT secret is deploy config. Sessions die if the secret rotates.
4. Display name is the GitHub login.

Public GitHub remotes still require this login. They do not require an App grant.

---

## Start a session

1. Signed-in user submits an HTTP git URL and a branch (default `main`).
2. Server canonicalizes the remote. If a live session exists for that pair:
   - same host login → return that session
   - anyone else → fail, no id
3. GitHub host/path remote: if the App cannot yet see the repo, send the user to the App install/update URL for that repo, then retry.
4. Mint an installation token (GitHub) or use the pasted token (other forges). Public clone may use no token.
5. HTTP clone into `{state-dir}/repos/{uuidv7}`. Hold the credential on the hub.
6. Open the CRDT hub. The host is admitted. Redirect to `/editor/{sessionID}`.

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
ask or auto-timer
    → singleflight
    → flush CRDT → git commit (author = host login) → git push
    → unlock
```

If push fails, keep the local commit, report the error, unlock. Do not start a second commit+push while one flight is running.

---

## Process death

Gone: hubs, CRDT, RAM credentials, working trees, knock queues, in-memory admits.

Still valid: git remotes, GitHub App install, JWT cookie (until expiry), deploy secrets.

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
| POST | `/sessions` | Create (clone) |
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

---

## Configuration

| Env | Role |
|-----|------|
| `JWT_SECRET` | Cookie and preauth signatures (required in production) |
| `GITHUB_APP_ID` | GitHub App id |
| `GITHUB_APP_CLIENT_ID` | OAuth client id |
| `GITHUB_APP_CLIENT_SECRET` | OAuth client secret |
| `GITHUB_APP_PRIVATE_KEY` | PEM for installation tokens |
| `GITHUB_APP_SLUG` | Install URL (`https://github.com/apps/{slug}/installations/new`) |
| `STATE_DIR` | Working trees + artifacts |
| `GO_ENV=development` | Allows JWT fallback |

SQLite/Postgres user and project tables are not used for identity or session lists. A driver may remain for a transition; it is not the product store.

---

## Module layout (added)

```text
internal/remote/      # canonicalize git HTTP URLs; parse GitHub owner/repo
internal/githubapp/   # OAuth, App JWT, installation token, install URL
internal/session/     # hub + live registry (uniqueness, admit, knock, RAM creds)
internal/git/         # clone, commit, push
```

---

## Risks

| Risk | Mitigation |
|------|------------|
| Unpushed work lost on nuke | Auto-push + singleflight; UI says Synced ≠ pushed |
| Installation token is the App | Never give it to browsers. Admit is identity-only |
| Guessable session URL | Knock off by default. Fail-on-duplicate does not leak the id |
| App not installed | Redirect to grant that repo, then retry clone |
| Two writers race create | Registry lock on `remote + branch` |

---

## Test strategy

| Layer | What |
|-------|------|
| remote | Canonical URL; GitHub owner/repo parse |
| session registry | Duplicate create fails; host retry returns same id; preauth admit; knock off rejects |
| git | Clone from a local remote; commit+push singleflight |
| githubapp | httptest for OAuth exchange and installation token |
| HTTP | GitHub-only login page; clone form creates session; second user cannot clone-join |

---

## Grill log (2026-08-24)

Forge-native → HTTP clone primitive, GitHub identity sugar → GitHub App grant-on-use, install token in RAM → clone creates UUIDv7 session → one live session per remote+branch, second create fails (no join) → host is first writer of this incarnation, no transfer → knock opt-in, auto-reject closes lobby only → preauth + admitted walk in → persist = ask + auto, singleflight → GitHub only, drop passwords → no user Ed25519 (deploy-key uniqueness).
