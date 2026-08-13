# gopub

> A self-hosted deployment / release system for ops teams · Go + Vue 3

<p align="center">
  <a href="README.md">中文</a> ·
  <b>English</b>
</p>

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Echo](https://img.shields.io/badge/Echo-v4-3d3d3d)
![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js&logoColor=white)
![MySQL](https://img.shields.io/badge/MySQL-5.7%2B-4479A1?logo=mysql&logoColor=white)
![License](https://img.shields.io/badge/License-Apache%202.0-blue)

---

## Table of Contents

- [What is gopub](#what-is-gopub)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Repository Layout](#repository-layout)
- [Release Pipeline](#release-pipeline)
- [Requirements](#requirements)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Docker](#docker)
- [Kubernetes](#kubernetes)
- [systemd](#systemd)
- [Packaging & CI](#packaging--ci)
- [Data Model](#data-model)
- [Public API](#public-api)
- [P2P Distribution](#p2p-distribution)
- [Security Notes](#security-notes)
- [Known Limitations](#known-limitations)
- [FAQ](#faq)
- [Development Guidelines](#development-guidelines)
- [Attribution & License](#attribution--license)

## What is gopub

gopub is a self-hosted web deployment system that turns "code from repository to target servers" into a configurable, auditable, rollback-capable pipeline.

The workflow: an operator registers a **project configuration** in the web UI (repository, target hosts, release directory, hook scripts, retained versions); a developer creates a **release ticket** picking a branch, tag, commit, or Jenkins build; on deploy the server runs *fetch → package → distribute → switch symlink → run hooks*, persisting every command and its output so failures are traceable and any retained version can be rolled back to in seconds.

Good fits:

- Legacy or not-yet-containerised projects deployed as **file trees** (PHP, static assets, Java archives).
- Releases that must land on **many hosts at once** with a consistent version.
- Teams that need an **audit trail** — who shipped which commit, when, and which commands ran.
- Setups that need to **expose deployment to other systems** through a JWT-authenticated REST API.

## Features

**Code sources**

- **Git** — pick a branch, tag, or commit; browse commit logs. `/api/get/task/changes` diffs a ticket against the project's last successful release (git projects only).
- **File** — pull an artifact straight from an HTTP(S) URL.
- **Jenkins** — fetch artifacts by build number, using `JenkinsUserName` / `JenkinsPwd` (env-var overridable).

**Release strategies**

- **Symlink switch** (`release_type=0`) — unpack into a per-version directory, then flip the symlink atomically. Instant activation, instant rollback.
- **Directory move** (`release_type=1`) — replace the target directory directly, for targets where symlinks don't work.
- **Version retention** (`keep_version_num`) — older versions are pruned automatically.
- **Exclusions** (`excludes`) — skip paths when packaging.

**Distribution channels**

- **SFTP** — the default: key-based SSH, pushed concurrently to all hosts via a goroutine pool (`grpool`).
- **P2P** — a built-in BitTorrent implementation; target hosts seed to each other, which drastically cuts egress bandwidth on the deploy host for large artifacts or large fleets. Enabled per project (`p2p`).
- **SSH algorithm tier** (`ssh_algorithm`) — `0` = modern, `1` = legacy, for talking to older OpenSSH daemons.

**Hook scripts** (one command per line, joined with `&&`, aborting on first failure)

| Hook | Runs on | Variables |
| --- | --- | --- |
| `pre_deploy` | deploy host, inside workspace | `{WORKSPACE}` `{HOSTS}` `{HOSTPORT}` `{ENV}` |
| `post_deploy` | deploy host, inside workspace | same |
| `pre_release` | target hosts, inside version dir | `{WORKSPACE}` `{VERSION}` `{HOSTS}` `{HOSTPORT}` `{ENV}` |
| `post_release` | target hosts, inside version dir | same |
| `last_deploy` | deploy host, after all hosts finish | above plus `{PROJECT_ID}` `{PROJECT_NAME}` `{TASK_ID}` `{TASK_LINKID}` |

**Host management**

- Maintain an `ip:port` list per project, or
- integrate **JumpServer** (`enableJumpserver`, built against the 1.5.3 API) and resolve node groups to IPs at deploy time, so scaling the fleet doesn't require editing project configs.

**Users and permissions**

- Local database accounts, or **LDAP** (`enableLdap`) with LDAP groups mapped to gopub roles.
- Three roles: `1` admin, `10` pre-release, `20` regular user. Regular users only see projects they've been granted (`group` table).
- Per-project **user lock** (`user_lock`, via `/api/get/conf/lock`) so two people don't edit the same config at once.

**Observability**

- Every command's text, duration, and result goes into the `record` table; the UI polls progress through seven stages (10/20/30/40/50/60/100).
- Failure details land in `task_err_log`.
- Release statistics charts (ECharts).
- Logs rotate daily and are kept for 30 days (`src/logs/run.log`).

**Integration**

- A `/v1` REST API with HS256 JWT auth, appid/appsecret token exchange, and a caller IP allowlist — callable from a CMDB, ticketing system, or CI.

## Tech Stack

| Layer | Choice |
| --- | --- |
| Backend | Go 1.25, [Echo v4](https://echo.labstack.com/), GORM 1.31 (MySQL driver), MySQL 5.7+ |
| Auth | console: `Authorization: TOKEN <auth_key>` (issued after bcrypt check, stored in `user.auth_key`); public API: [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) HS256 |
| Remote execution | `golang.org/x/crypto/ssh`, `pkg/sftp`, `sshexec`, `grpool` worker pool |
| Integrations | `gojenkins`, `go-ldap/ldap/v3`, JumpServer REST API |
| P2P | in-tree BitTorrent implementation (`src/library/p2p`), `httprouter` management endpoints, `seelog` |
| Frontend | Vue 3.5, Vue Router 4, Vuex 4, Vite 8, Element Plus 2, ECharts 6, Axios, Sass |
| Deployment | multi-stage Docker, systemd, Kubernetes, GitHub Actions release |

## Architecture

```text
                       ┌──────────────────────────────┐
   Browser ──HTTP──►    │  Echo HTTP Server (:8192)    │
   Other systems ─/v1─► │  ├─ routes   src/routers     │
                        │  ├─ handlers src/controllers │
                        │  ├─ TOKEN / JWT auth         │
                        │  └─ template views/index.tpl │
                        └───────────┬──────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
             ┌────────────┐  ┌──────────────┐  ┌──────────────┐
             │ GORM/MySQL │  │ release core │  │ external     │
             │ project    │  │ library/     │  │ Git / Jenkins│
             │ task       │  │ components   │  │ LDAP         │
             │ record ... │  └──────┬───────┘  │ JumpServer   │
             └────────────┘         │          └──────────────┘
                                    │
                 ┌──────────────────┴──────────────────┐
                 ▼                                     ▼
        ┌────────────────────┐            ┌────────────────────┐
        │ concurrent SFTP    │            │ P2P server (45002) │
        │ over SSH (grpool)  │            │ + agents seeding   │
        └─────────┬──────────┘            └─────────┬──────────┘
                  └───────────────┬──────────────────┘
                                  ▼
               target hosts: versioned dirs + symlink switch
```

## Repository Layout

```text
.
├── src/                          # Go backend and runtime assets
│   ├── main.go                   # Echo bootstrap, templates, signals, P2P server start
│   ├── controllers/              # HTTP handlers
│   │   ├── api/                  # /v1 public REST API (JWT)
│   │   ├── conf/                 # project CRUD, copy, lock, host groups
│   │   ├── task/                 # release tickets, rollback, charts
│   │   ├── walle/                # deploy execution, Git/Jenkins queries, env checks
│   │   ├── p2p/                  # P2P agent dispatch and status
│   │   ├── record/ user/ other/  # deploy records, users, misc
│   │   ├── base.go               # request context, unified JSON responses
│   │   └── login*.go register.go changepasswd.go
│   ├── library/
│   │   ├── components/           # release engine (folder/git/file/task/base)
│   │   ├── ssh/                  # SSH/SFTP/P2P transfer, algorithm tiers
│   │   ├── p2p/                  # BitTorrent implementation (server/agent/p2p/flowctrl)
│   │   ├── db/                   # GORM init and pooling
│   │   ├── config/               # app.conf parser (beego-ini compatible semantics)
│   │   ├── logger/               # daily-rotating logs
│   │   ├── cache/ paging/        # in-memory cache, pagination helpers
│   │   ├── ldap/ jumpserver/     # external integrations
│   │   ├── publog/ common/       # deploy logging, shared utilities
│   ├── models/                   # GORM models, AutoMigrate, seed data
│   ├── routers/router.go         # all route registration (incl. CORS)
│   ├── tasks/                    # background jobs (P2P agent health check)
│   ├── conf/app.conf             # main configuration file
│   ├── agent/                    # P2P agent binary, server.json / agent.json
│   ├── static/  views/index.tpl  # Vite output and entry template (build artefacts, not in git)
│   └── logs/                     # runtime and task logs
├── frontend/                     # Vue 3 + Vite console
│   ├── src/pages/                # conf / task / user / p2p / charts / home
│   ├── src/components/ store/ router/ request/ common/
│   └── vite.config.js            # emits to ../src/static and refreshes index.tpl
├── Dockerfile                    # three stages (golang / node / alpine)
├── control                       # build / start / stop / init / pack helper
├── gopub.service                 # systemd unit
├── gopub-kubernetes.yml          # Deployment + Service (MySQL sidecar example)
└── .github/workflows/
    ├── ci.yml                    # push / PR → build, vet, test
    └── release.yml               # tag v* → package and publish a Release
```

## Release Pipeline

`/api/get/walle/release` kicks off an asynchronous job (`src/controllers/walle/release.go:108`). Each step writes to `record` and advances the progress code:

| Code | Step | What happens |
| --- | --- | --- |
| 10 | `InitLocalWorkspace` / `InitRemoteVersion` | mint version `YYYYMMDD-HHMMSS`, create local workspace and remote version dir |
| 20 | `PreDeploy` | local pre-deploy hook |
| 30 | `UpdateToVersion` | fetch code at the requested version per `repo_type` (git / file / jenkins) |
| 40 | `PostDeploy` | local post-deploy hook (build & dependency install usually live here) |
| 50 | `CopyFiles` | package (optional gzip, honouring `excludes`) and distribute via SFTP or P2P |
| 60 | `UpdateRemoteServers` | on each target: `pre_release` → symlink switch / dir move → `post_release` |
| — | `LastDeploy` | once every host is done, run wrap-up commands locally |
| 100 | `CleanUpLocal` / `CleanUpReleasesVersion` | clean the workspace, prune versions beyond `keep_version_num` |

Workspace path: `{deploy_from}/{env}/{project}-{version}`, where `env` comes from the project `level` (`1`=test, `2`=simu, `3`=prod).

**Rollback** — when a ticket's `action != 0`, `rollBackHandling` runs `UpdateRemoteServers` only, repointing the symlink at a previously deployed version without re-fetching code. That's why rollback is near-instant; it requires the version to still be within `keep_version_num`.

**Ticket status** (`task.status`, labels in `src/controllers/task/list.go:56`):

| Value | Meaning |
| --- | --- |
| `0` / `1` | newly submitted |
| `2` | audit rejected |
| `3` | deployed successfully |
| `4` | deploy failed |

While `is_run=1`, the list page shows "deploying" instead.

**Concurrency guards** — a ticket with `is_run=1` cannot be triggered again; tickets in `status=2` (audit rejected) or `3` (deployed) cannot be deployed again; non-owners who aren't admins cannot deploy someone else's ticket (`release.go:38`).

## Requirements

| Component | Version | Notes |
| --- | --- | --- |
| Go | >= 1.25 | see `go.mod` |
| Node.js | >= 20.19.0 | `engines` in `frontend/package.json` |
| npm | >= 10 | same |
| MySQL | 5.7+ or compatible | first run does `CREATE DATABASE IF NOT EXISTS` and creates tables |
| Git | any | the deploy host must be able to clone your repos |
| SSH | OpenSSH | the deploy host must reach targets with key auth |

Builds need access to a Go module proxy and the npm registry.

## Getting Started

### 1. Set up key-based SSH

gopub logs into target hosts over SSH, so the OS user running gopub needs a key pair whose public key is in the **release user's** (`release_user`) `~/.ssh/authorized_keys` on every target:

```shell
ssh-keygen -t rsa -b 4096 -N '' -f ~/.ssh/id_rsa
ssh-copy-id -i ~/.ssh/id_rsa.pub deploy@TARGET_HOST
```

### 2. Configure the database

Edit `src/conf/app.conf` and fill in the section matching your run mode (`[dev]` / `[prod]` / `[docker]`):

```ini
runmode = prod          # selects which section wins

[prod]
HttpPort  = 8192
mysqluser = "root"
mysqlpass = "your-password"
mysqlhost = "127.0.0.1"
mysqlport = 3306
mysqldb   = "go_pub"
SecretKey = "replace-with-a-long-random-string"
```

> Configuration is only loaded from `conf/app.conf` relative to the working directory, or next to the executable (`src/library/config/config.go:43`). There is no `app.local.conf` override mechanism — edit `src/conf/app.conf` directly, and take care not to commit real credentials.

### 3. Build the frontend

`src/static` and `src/views/index.tpl` are Vite build artefacts and are **not tracked in git**, so a fresh clone has no frontend at all — you must build it once before the console renders anything. The Docker build (`Dockerfile`, node stage) and the Release workflow (`.github/workflows/release.yml`) both run this step themselves, so this only applies to local checkouts.

```shell
cd frontend
npm ci
npm run build        # emits to ../src/static and refreshes ../src/views/index.tpl
```

Or, from the repository root, `./control webbuild` — it runs `npm ci` on first use and then `npm run build`.

Dev server (proxies `/api` to `127.0.0.1:8192` by default):

```shell
cd frontend && npm run dev
```

### 4. Build and run the backend

```shell
./control buildall   # frontend + backend; use this after a fresh clone
./control build      # gofmt + go build only, produces src/gopub
./control init       # create schema and seed the admin user (same as src/gopub -syncdb)
./control start      # start in background, pid written to gopub.pid
./control status
./control tail       # follow src/logs/stdout.log
./control stop
```

Open `http://127.0.0.1:8192`. The seeded admin username is `admin`; its initial bcrypt hash is hard-coded in `src/models/AdminInit.go` (inherited from upstream) — **change the password immediately after first login**.

`control` subcommands: `build | webbuild | buildall | pack | start | stop | kill | restart | reload | status | run | rundocker | init | tail | docs | beerun | sslkey`.

> `webbuild` is deliberately kept out of `build`: the golang stage in `Dockerfile` runs `./control build` in an image that has no Node.js, so folding the frontend build into `build` would break the image. `pack` calls `buildall`, since `src/static` is no longer in git and would otherwise be packaged empty.

## Configuration

The file is ini-formatted. Keys are case-insensitive (lower-cased internally), and lookups check the **current `runmode` section first, then fall back to the global section**.

### Server

| Key | Default | Notes |
| --- | --- | --- |
| `appname` | `gopub` | application name |
| `runmode` | `prod` | `dev` / `prod` / `docker`; selects the active section |
| `HttpAddr` | `0.0.0.0` | listen address |
| `HttpPort` | `8192` | listen port (the global `httpport = 8080` is overridden by every mode section) |
| `EnableGzip` | `true` | gzip responses |
| `AccessLogs` | per section | access logging |
| `Graceful` | `false` | when `true`, drain in-flight requests on shutdown (10s timeout) |
| `SshPort` | `22` | SSH port used for target hosts |
| `SecretKey` | `1234` | **JWT signing key — must be changed** |
| `SessionOn` / `SessionGCMaxLifetime` / `SessionCookieLifeTime` | `true` / `86400` / `86400` | beego leftovers; no code reads them after the Echo migration, so changing them has no effect |
| `AutoRender` / `CopyRequestBody` / `EnableDocs` / `EnableHTTP` / `HttpsPort` / `EnableAdmin` / `AdminAddr` / `AdminPort` | — | also beego leftovers; re-readable request bodies are now unconditional in `src/controllers/base.go` |

### Database

| Key | Notes |
| --- | --- |
| `mysqluser` / `mysqlpass` / `mysqlhost` / `mysqlport` / `mysqldb` | connection parameters |
| `db_max_idle_conn` / `db_max_open_conn` | pool sizes (defaults 30 / 100) |

When `runmode=docker`, the env vars `MYSQL_USER` / `MYSQL_PASS` / `MYSQL_HOST` / `MYSQL_PORT` / `MYSQL_DB` take precedence over the file (`src/library/db/db.go:91`). **They have no effect outside docker mode.**

### Jenkins / mail / P2P

| Key | Notes |
| --- | --- |
| `JenkinsUserName` / `JenkinsPwd` | Jenkins credentials; same-named env vars win (`src/main.go:54`) |
| `emailUsername` / `emailPwd` / `emailHost` / `emailPort` | notification mailbox |
| `AgentDir` / `AgentDestDir` | local P2P agent dir and the dir it is pushed to on targets |

### LDAP

| Key | Notes |
| --- | --- |
| `enableLdap` | `true` authenticates against LDAP; `false` uses local accounts |
| `ldapHost` / `ldapPort` | server |
| `ldapPeopleDn` / `ldapPeopleDnTpl` | user DN and bind template, `{uid}` placeholder |
| `ldapGroupDn` / `ldapGroupFilter` | group lookup, supports `{UidNumber}` `{uid}` `{cn}` `{sn}` |
| `ldapGroupName2roleid_gopubAdmin` / `_gopubPre` / `_gopubSingle` | LDAP group → role (1 / 10 / 20); the three groups must exist in LDAP |

### JumpServer

| Key | Notes |
| --- | --- |
| `enableJumpserver` | lets projects pick a host group and resolve IPs at deploy time |
| `jumpserver` / `jump_username` / `jump_password` | endpoint and credentials |
| `jump_auth_api` / `jump_grouplist_api` / `jump_groupid2ips_api` | API paths (defaults target JumpServer 1.5.3) |

## Docker

```shell
docker build -t gopub .

docker run --name gopub \
  -e MYSQL_HOST=127.0.0.1 \
  -e MYSQL_PORT=3306 \
  -e MYSQL_USER=root \
  -e MYSQL_PASS=123456 \
  -e MYSQL_DB=walle \
  -p 8192:8192 \
  --restart always -d gopub:latest
```

About the image:

- Three stages — `golang:1.25-alpine` builds the backend, `node:22-alpine` builds the frontend, `alpine:3.22` is the runtime.
- The runtime ships `bash git openssh curl wget tzdata`, timezone pinned to `Asia/Shanghai`.
- The build runs `ssh-keygen` to create `/root/.ssh/id_rsa` and prints the public key into the build log — **add that key to your targets' `authorized_keys`**.
- Entrypoint is `./control rundocker`, i.e. started with `-docker`, which auto-creates the schema and enables the `MYSQL_*` env overrides.

> For production, mount `/root/.ssh`, `src/conf`, and `src/logs` as volumes — otherwise recreating the container loses the key, the config, and the logs.

## Kubernetes

```shell
kubectl apply -f gopub-kubernetes.yml
```

`gopub-kubernetes.yml` is an **example** manifest: one Deployment (gopub plus a MySQL 5.7 sidecar) and a NodePort Service on `8192`. Before production, at minimum:

- point `image` at your own registry;
- move passwords out of `env` into a `Secret`;
- replace the MySQL sidecar + `hostPath` with a real stateful service or managed database;
- tighten `securityContext.privileged: true`;
- replace the sample `hostAliases` entries with your GitLab / Jenkins addresses.

## systemd

```shell
# assumes /www/server/gopub, the path baked into gopub.service
cp gopub.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now gopub
systemctl status gopub
```

The unit is `Type=forking` and simply wraps `control start|restart|stop`. Adjust the three `Exec*` paths if you deploy elsewhere.

## Packaging & CI

Local packaging:

```shell
./control pack       # runs buildall, then writes ../gopub.tar.gz with control, service, binary, conf, views, static, agent
```

Continuous checks (`.github/workflows/ci.yml`): every branch push and pull request runs `gofmt -l src/`, `go build ./...`, `go vet ./...`, and `go test ./...` on the backend, plus `npm ci` + `npm run build` on the frontend. `src/routers/router_test.go` is the one to watch — a new route that forgets `RequireLogin` fails here.

CI packaging (`.github/workflows/release.yml`): pushing a `v1.2.3`-style tag triggers *build frontend → `go build -trimpath -ldflags "-s -w"` → tar the same manifest as `control pack` into `gopub-<tag>.tar.gz` → create a GitHub Release with auto-generated notes and attach the archive*.

```shell
git tag v1.2.3 && git push origin v1.2.3
```

## Data Model

`./control init` (or the first `-docker` start) creates the database and `AutoMigrate`s these tables:

| Table | Purpose |
| --- | --- |
| `user` | users: `role`, `from_ldap` flag, bcrypt password hash |
| `group` | project grants (`project_id` + `user_id` + `type`) |
| `project` | project config: repo, hosts, release dir, strategy, hooks, retention, P2P/gzip/SSH-algorithm flags |
| `task` | release tickets: branch, commit, file list, status, rollback flag, running flag |
| `record` | per-command execution log (command, duration, result, progress code) — powers the progress bar |
| `task_err_log` | failure details |
| `session` | legacy beego session table; still created, but no code reads or writes it |
| `api_system` | public API appid / appsecret / IP allowlist |
| `migration` | migration bookkeeping |

## Public API

Everything under `/v1` is meant for external callers and uses HS256 JWT signed with the configured `SecretKey`.

### 1. Exchange credentials for a token

```http
GET /v1/token?appid=<appid>&appsecret=<appsecret>
```

The server checks three things: the appid exists, the appsecret matches, and **the caller's IP is in that appid's allowlist** (`api_system.ip`, comma-separated).

Success:

```json
{ "access_token": "eyJhbGciOiJIUzI1NiIs...", "expires_in": "1750000000" }
```

Failure:

```json
{ "errcode": "100", "errmsg": "appid不存在 " }
```

Tokens last one hour, carry the appid in `iss`, and `expires_in` is the absolute Unix expiry timestamp (not a remaining-seconds value).

> `/v1` credentials are **system-level**: the token binds only to an appid and is tied to no gopub user, so it is **not subject to [object-level permissions](#object-level-permissions)** — the holder can read and write every project and ticket. That is by design for an integration API; the gate is the appsecret plus the IP allowlist. Treat it as a system credential and don't hand it out to individuals.

### 2. Release ticket endpoints

Send the token in the `Authorization` header — **the bare token, with no `Bearer ` prefix**:

| Method and path | Purpose |
| --- | --- |
| `POST /v1/task` | create a release ticket |
| `GET /v1/task` | list tickets by project name and time range |
| `GET /v1/task/:id` | fetch one ticket |
| `PUT /v1/task/:id` | update a ticket |
| `DELETE /v1/task/:id` | delete a ticket |

```shell
TOKEN=$(curl -s "http://127.0.0.1:8192/v1/token?appid=1&appsecret=xxx" | jq -r .access_token)
curl -H "Authorization: $TOKEN" "http://127.0.0.1:8192/v1/task/1"
```

Error codes: `102` missing token, `103` token rejected (expired, bad signature, or algorithm other than HS256).

### 3. Internal endpoints

The console talks to the `/api/get/*` and `/api/post/*` families (project config, tickets, Git/Jenkins queries, deploy execution, P2P, records, users). The full list lives in `src/routers/router.go`.

Their auth differs from `/v1`: on successful login the server issues a 32-character random string (`crypto/rand`), stores it in `user.auth_key`, and returns it; the frontend then sends `Authorization: TOKEN <auth_key>` and the server looks the user up by that key (`userByToken` at `src/controllers/auth.go:78`).

Credential lifecycle is centralised in `src/models/user_authkey.go`:

- **Lifetime** — configured via `authKeyLifetime` (seconds, default `604800`, i.e. 7 days); the expiry instant lives in `user.auth_key_expire_at`.
- **Sliding renewal** — any request extends it; the credential lapses only after `authKeyLifetime` of inactivity. To avoid a write per request, the `UPDATE` only fires once less than half the lifetime remains.
- **Multi-device** — logging in again reuses an existing unexpired `auth_key` and merely extends its expiry, so signing in on another machine does not kick out the first.
- **Revocation** — `POST /logout` and a password change clear `auth_key` and set `auth_key_expire_at` to NULL, invalidating every session for that account immediately.

Rows with a NULL `auth_key_expire_at` count as logged out, so everyone must sign in once after upgrading to this version — that is precisely how the older, non-expiring credentials get revoked.

#### Object-level permissions

Beyond being logged in, there is a second layer: whether a user may read or write a given project / ticket is decided in one place, `src/controllers/perm.go`, with the same semantics as the project list filter (`src/controllers/conf/mylist.go:30-35`) — **if you can see it in the list, you can act on it**:

| `user.role` | Meaning | Projects it may act on |
| --- | --- | --- |
| `1` | Administrator | All |
| `10` | Pre-release user | Only `level=2` (simu) |
| `20` | Single-project user | Only those granted in the `group` table |
| Anything else (incl. `0`) | Unknown role | None |

Enforcement is centralised at context load: `ctx.Project` / `ctx.Task` are only attached when the current user actually has permission (`src/controllers/base.go:55-64`); otherwise they stay `nil` and each handler's existing nil check rejects the request. Write endpoints that bypass the context and look records up by ID themselves (editing project config or tickets, delete, lock, bulk flush) call `CanAccessProject` / `CanAccessTask` explicitly. Every decision is made against the **stored record**, never against fields from the request body — trusting a submitted `level` or `project_id` would defeat the check.

> ⚠️ **Upgrade note:** if you authenticate via LDAP and some groups have no `ldapGroupName2roleid_*` entry in `app.conf`, those users end up with `role = 0`. Before this change they could see **every project** in the list; after it they have **no project access at all**. Run `SELECT id, username, role FROM user WHERE role NOT IN (1, 10, 20);` before upgrading to confirm there are no such accounts, or add the missing role mappings.

## P2P Distribution

Pushing a large artifact to many hosts over SFTP saturates the deploy host's uplink. gopub ships a BitTorrent implementation for that case:

- The **server** runs inside the gopub process (started by `init_sever.Start()` at `src/main.go:126` outside docker mode) and handles seeding, task creation, and task queries.
- The **agent** is a standalone binary pushed to targets (`src/agent/`); agents exchange pieces with each other.
- Default ports: management `45003`, data `45002`, with matching management/data ports on the agent side. Speed limit (MBps), memory cache size, and max concurrent tasks are configurable — see `src/agent/README.md` and `src/agent/server.json`.
- The console has an agent-status page, and `src/tasks/check_p2p_agent.go` polls agent liveness.
- Turning on a project's `p2p` flag makes the `CopyFiles` stage use P2P instead of SFTP.

## Security Notes

⚠️ This system holds SSH keys for your fleet and can run arbitrary commands on it — treat it like a bastion host.

- **Change `SecretKey`.** It signs the JWTs, and the committed default is `1234`.
- **Change the default admin password.** `admin`'s seeded hash is a publicly known upstream value.
- **Don't commit real credentials.** `src/conf/app.conf` still carries sample/legacy values (database password, internal IPs, mailbox). Override them at deploy time, and consider removing the file from version control or injecting config via environment.
- **Fields like `repo_password` are stored in plaintext** in MySQL — restrict database access accordingly.
- **Hooks and deploy commands execute on targets as `release_user`**, so anyone with project permission effectively has command execution there. There is **no working approval step** (see [Known Limitations](#known-limitations)), so the project grants in the `group` table are the only gate — grant sparingly.
- **CORS is currently `AllowOrigins: ["*"]`** (`src/routers/router.go:27`) — tighten it before exposing the service.
- **Prefer HTTPS in production.** The HTTPS-enforcement block in `src/controllers/api/base_api.go` is commented out; enable it or terminate TLS at a reverse proxy.
- Give `release_user` write access only to the directories it needs; don't run as root.

## Known Limitations

- Test coverage is still lowish: the repository has four test files — `src/library/ssh/remote_test.go` (SSH algorithm tiers), `src/routers/router_test.go` (route inventory and auth enforcement), `src/library/common/shell_test.go` (shell quoting and git ref validation), and `src/controllers/perm_test.go` (the object-level permission matrix); every other package reports `no test files`, and there are no frontend tests.
- Four project fields are inert: `post_release_together` ("run once after every host finishes"), `gzip`, `audit`, and `view_history`. They exist on `models.Project` and have defaults in the frontend form, but nothing on the backend reads them, so setting them changes no behaviour. Use `last_deploy` for wrap-up commands, and `/api/get/task/changes` for diffs (it does not consult `view_history`). (`EnableGzip` in `app.conf` is unrelated — that's HTTP response compression and works.)
- **There is no approval step.** The `status=2` ("audit rejected") state exists, but no code ever sets a ticket to 2 and there is no approval endpoint — anyone with project permission can deploy directly.
- What the unauthenticated allowlist exposes is not permission-filtered: anyone can pass a `taskId` / `projectId` to those four public read-only endpoints and get back the **display fields** of a ticket or project (title, status, branch, project name, environment) plus the **full release record** (every command and its output, from the `record` table). Sensitive fields are redacted — projects are reduced to the `publicProject` allowlist (no repo password, host list, or deploy commands) and tickets have `hosts` stripped — but the command output itself remains public.
- Four internal endpoints are **deliberately unauthenticated**: `/api/get/task/list`, `/api/get/task/get`, `/api/get/conf/get`, and `/api/get/record/list`. They back the login-free log viewer (`searchtaskList` / `searchtaskRelease`); the allowlist is spelled out at `src/routers/router.go:44-47` and registered in `noAuthRoutes` in `src/routers/router_test.go`. This means **deploy ticket lists, project config, and release records are readable without credentials** — restrict them at a reverse proxy before exposing the service. (Every other internal route is gated by `RequireLogin` at `src/controllers/auth.go:42`; account creation/deletion and the user list additionally require `RequireAdmin` — see `src/routers/router.go:54` and `:107`.)
- Element Plus is registered wholesale via `app.use(ElementPlus)`. It builds into its own `element-plus` chunk (~970 kB, 311 kB gzipped) and is a first-paint dependency. Switching to on-demand imports would shrink it considerably, but that also means taking over the 54 call sites using globals like `this.$message` / `this.$confirm`.
- The JumpServer integration targets the 1.5.3 API; newer JumpServer releases need adapting.

## FAQ

**Q: Startup fails with "cannot find conf/app.conf".**
Configuration is resolved relative to the **working directory**. Start via `./control start` (which `cd`s into `src/`), or `cd src/` before running the binary.

**Q: Is the port 8080 or 8192?**
The global section says `httpport = 8080`, but `[dev]`, `[prod]`, and `[docker]` all set `HttpPort = 8192`, and section values win — so it's **8192**.

**Q: I set `MYSQL_HOST` and nothing changed.**
`MYSQL_*` env vars only override the file when `runmode=docker`. For non-container deployments, edit `app.conf`.

**Q: A deploy hangs at 50 (file distribution).**
Use the project page's detection feature (`/api/get/walle/detectionssh`) to verify key-based SSH and target directory permissions. For old sshd versions, switch the project's SSH algorithm tier to `legacy`.

**Q: The version I want isn't available for rollback.**
Retention is capped by `keep_version_num`; anything beyond that was pruned at the end of a previous deploy.

## Development Guidelines

- Format Go with `gofmt` (`./control build` runs `gofmt -w src/` first). Keep package names short, lowercase, matching their directory.
- Controllers are grouped by feature (`task` / `conf` / `api` / `walle` / `user` / `p2p` / `record` / `other`); add handlers beside related behaviour.
- Dependencies are managed with Go modules — **do not reintroduce `vendor/`**.
- Frontend is Vue 3 + Vuex 4 + Vue Router 4 + Element Plus; store modules in `frontend/src/store/`, routes in `frontend/src/router/`, API path constants in `frontend/src/common/port_uri/`. Prefer ESM over CommonJS.
- Add Go tests as `*_test.go` beside the package under test, favouring table-driven cases. Frontend tests as `*.spec.js` / `*.test.js`, with the npm script documented in `package.json`.
- Keep commits short and imperative. PRs should describe the behaviour change, list manual verification commands, call out config or database impact, and include screenshots for UI changes. Don't mix dependency upgrades with feature work.

Verification commands:

```shell
go build ./...
go vet ./...
go test ./...
cd frontend && npm run build
cd frontend && npm audit --audit-level=high
```

## Attribution & License

This repository is a derivative work of [linclin/gopub](https://github.com/linclin/gopub), with changes including the Beego → Echo migration, a JWT library replacement, SSH algorithm tiers, and a release packaging pipeline. See [NOTICE](NOTICE) and the git history for the full record.

Licensed under the [Apache License 2.0](LICENSE). Original copyright belongs to linclin and contributors (Copyright 2018 linclin); modifications Copyright 2026 lnatpunblhna.
