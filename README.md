# gopub

> 面向运维发布场景的一站式代码发布系统 · Go + Vue 3

<p align="center">
  <b>中文</b> ·
  <a href="README_EN.md">English</a>
</p>

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Echo](https://img.shields.io/badge/Echo-v4-3d3d3d)
![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js&logoColor=white)
![MySQL](https://img.shields.io/badge/MySQL-5.7%2B-4479A1?logo=mysql&logoColor=white)
![License](https://img.shields.io/badge/License-Apache%202.0-blue)

---

## 目录

- [项目简介](#项目简介)
- [核心特性](#核心特性)
- [技术栈](#技术栈)
- [系统架构](#系统架构)
- [目录结构](#目录结构)
- [发布流程详解](#发布流程详解)
- [环境要求](#环境要求)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [Docker 部署](#docker-部署)
- [Kubernetes 部署](#kubernetes-部署)
- [systemd 托管](#systemd-托管)
- [发布制品与 CI](#发布制品与-ci)
- [数据模型](#数据模型)
- [对外 API](#对外-api)
- [P2P 分发](#p2p-分发)
- [安全须知](#安全须知)
- [已知限制](#已知限制)
- [常见问题](#常见问题)
- [开发约定](#开发约定)
- [致谢与许可](#致谢与许可)

## 项目简介

gopub 是一个自托管的 Web 发布系统，把「代码从仓库到目标服务器」这条链路做成可配置、可审计、可回滚的流水线。

典型使用方式是：运维在 Web 界面里为每个项目登记一份**项目配置**（仓库地址、目标机器列表、发布目录、钩子脚本、保留版本数），开发者按需创建**上线单**（选择分支 / tag / commit / Jenkins 构建号），点击部署后由服务端串行执行「拉取 → 打包 → 分发 → 切换软链 → 执行钩子」，全过程命令与输出落库，失败可查、可基于历史版本一键回滚。

适合的场景：

- 传统 PHP / 静态资源 / Java 包等**基于文件目录发布**的项目，尤其是尚未容器化的存量业务；
- 需要**多机同时发布**并保证版本一致的场景；
- 需要**发布记录审计**（谁在什么时候发了哪个 commit、执行了哪些命令）的团队；
- 需要把发布能力**开放给外部系统调用**（提供 JWT 鉴权的 REST API）。

## 核心特性

**代码来源**

- **Git**：分支 / tag / commit 三种粒度选择，支持查看提交记录；`/api/get/task/changes` 可对比本次上线单与该项目上一次成功上线之间的文件差异（仅 git 类型项目可用）。
- **文件**：直接从 HTTP(S) 地址拉取制品包。
- **Jenkins**：按 Job 的构建号取制品，登录凭据来自 `JenkinsUserName` / `JenkinsPwd`（支持环境变量覆盖）。

**发布方式**

- **软链切换**（`release_type=0`）：新版本解包到独立版本目录，最后原子地切换软链，秒级生效、秒级回滚。
- **移动目录**（`release_type=1`）：直接替换目标目录，适配不支持软链的场景。
- **保留版本数**（`keep_version_num`）：超出份数的历史版本自动清理。
- **排除规则**（`excludes`）：打包时排除指定路径。

**分发通道**

- **SFTP**：默认通道，基于 SSH 密钥免密登录，通过协程池（`grpool`）并发推送到多台机器。
- **P2P**：内置 BitTorrent 协议实现，大包 / 多机场景下由各节点互相做种，显著降低发布机出口带宽压力。可在项目上按需开启（`p2p`）。
- **SSH 算法档位**（`ssh_algorithm`）：`0` = modern（现代算法集），`1` = legacy（兼容老旧 sshd 的算法集），用于对接老版本 OpenSSH 的目标机。

**钩子脚本**（每行一条命令，`&&` 串联执行，失败即中断）

| 钩子 | 执行位置 | 可用变量 |
| --- | --- | --- |
| `pre_deploy` | 发布机本地，工作空间内 | `{WORKSPACE}` `{HOSTS}` `{HOSTPORT}` `{ENV}` |
| `post_deploy` | 发布机本地，工作空间内 | 同上 |
| `pre_release` | 目标机，版本目录内 | `{WORKSPACE}` `{VERSION}` `{HOSTS}` `{HOSTPORT}` `{ENV}` |
| `post_release` | 目标机，版本目录内 | 同上 |
| `last_deploy` | 发布机本地，全部机器完成后 | 上述变量 + `{PROJECT_ID}` `{PROJECT_NAME}` `{TASK_ID}` `{TASK_LINKID}` |

**主机管理**

- 直接在项目里维护 `ip:port` 列表；
- 或对接 **JumpServer**（`enableJumpserver`，适配 1.5.3 版本 API），按节点分组动态获取 IP，扩容缩容不用改项目配置。

**用户与权限**

- 本地数据库账号，或对接 **LDAP**（`enableLdap`）统一认证，LDAP 用户组映射到 gopub 角色；
- 三级角色：`1` 管理员、`10` 预发布、`20` 普通用户，普通用户只能看到自己被授权的项目（`group` 表）；
- 项目支持**用户锁定**（`user_lock`，`/api/get/conf/lock`），避免多人同时改同一份配置。

**可观测性**

- 每条命令的执行内容、耗时、返回值写入 `record` 表，页面实时刷新进度（10/20/30/40/50/60/100 七个阶段）；
- 失败堆栈写入 `task_err_log`；
- 上线单统计图表（ECharts）；
- 日志按天切割，保留 30 天（`src/logs/run.log`）。

**对外集成**

- `/v1` REST API，JWT（HS256）鉴权 + appid/appsecret 换 token + 调用方 IP 白名单，可供 CMDB、工单系统、CI 调用。

## 技术栈

| 层次 | 选型 |
| --- | --- |
| 后端 | Go 1.25、[Echo v4](https://echo.labstack.com/)、GORM 1.31（MySQL 驱动）、MySQL 5.7+ |
| 鉴权 | 控制台：`Authorization: TOKEN <auth_key>`（bcrypt 校验后签发，存 `user.auth_key`）；对外 API：[golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) HS256 |
| 远端执行 | `golang.org/x/crypto/ssh`、`pkg/sftp`、`sshexec`、`grpool` 协程池 |
| 集成 | `gojenkins`（Jenkins）、`go-ldap/ldap/v3`（LDAP）、JumpServer REST API |
| P2P | 自带 BitTorrent 实现（`src/library/p2p`）+ `httprouter` 管理接口 + `seelog` 日志 |
| 前端 | Vue 3.5、Vue Router 4、Vuex 4、Vite 8、Element Plus 2、ECharts 6、Axios、Sass |
| 部署 | Docker 多阶段构建、systemd、Kubernetes、GitHub Actions Release |

## 系统架构

```text
                       ┌──────────────────────────────┐
   浏览器 ──HTTP──►     │  Echo HTTP Server (:8192)    │
   外部系统 ──/v1──►    │  ├─ 路由 src/routers         │
                       │  ├─ 控制器 src/controllers    │
                       │  ├─ TOKEN / JWT 鉴权          │
                       │  └─ 模板 views/index.tpl      │
                       └───────────┬──────────────────┘
                                   │
                    ┌──────────────┼───────────────┐
                    ▼              ▼               ▼
             ┌────────────┐  ┌──────────┐   ┌─────────────┐
             │ GORM/MySQL │  │ 发布引擎  │   │ 外部系统     │
             │ project    │  │ library/ │   │ Git / Jenkins│
             │ task       │  │ components│  │ LDAP         │
             │ record ... │  └────┬─────┘   │ JumpServer   │
             └────────────┘       │         └─────────────┘
                                  │
                 ┌────────────────┴─────────────────┐
                 ▼                                  ▼
        ┌─────────────────┐                ┌──────────────────┐
        │ SFTP + SSH 并发  │                │ P2P Server(45002)│
        │ (grpool 协程池)  │                │ + Agent 互相做种  │
        └────────┬────────┘                └────────┬─────────┘
                 └──────────────┬────────────────────┘
                                ▼
                     目标服务器：版本目录 + 软链切换
```

## 目录结构

```text
.
├── src/                          # Go 后端与运行时资源
│   ├── main.go                   # 入口：Echo 初始化、模板、信号处理、启动 P2P server
│   ├── controllers/              # HTTP 控制器
│   │   ├── api/                  # /v1 对外 REST API（JWT 鉴权）
│   │   ├── conf/                 # 项目配置增删改查、复制、锁定、分组
│   │   ├── task/                 # 上线单、回滚、统计图表
│   │   ├── walle/                # 发布执行、Git/Jenkins 查询、环境检测
│   │   ├── p2p/                  # P2P agent 下发与状态查询
│   │   ├── record/ user/ other/  # 发布记录、用户、杂项
│   │   ├── base.go               # 请求上下文、统一 JSON 响应
│   │   └── login*.go register.go changepasswd.go
│   ├── library/
│   │   ├── components/           # 发布引擎核心（folder/git/file/task/base）
│   │   ├── ssh/                  # SSH/SFTP/P2P 传输封装、算法档位
│   │   ├── p2p/                  # BitTorrent 实现（server / agent / p2p / flowctrl）
│   │   ├── db/                   # GORM 初始化与连接池
│   │   ├── config/               # app.conf 解析（兼容 beego ini 语义）
│   │   ├── logger/               # 按天切割日志
│   │   ├── cache/ paging/        # 内存缓存、分页组件
│   │   ├── ldap/ jumpserver/     # 外部系统对接
│   │   ├── publog/ common/       # 发布日志、通用工具
│   ├── models/                   # GORM 模型 + AutoMigrate + 初始化数据
│   ├── routers/router.go         # 全部路由注册（含 CORS）
│   ├── tasks/                    # 后台任务（P2P agent 健康检查）
│   ├── conf/app.conf.example     # 配置模板（入库）；app.conf 由 control 生成且不入库
│   ├── agent/                    # P2P agent 二进制与 server.json / agent.json
│   ├── static/  views/index.tpl  # Vite 构建产物与入口模板
│   └── logs/                     # 运行日志、任务日志
├── frontend/                     # Vue 3 + Vite 前端
│   ├── src/pages/                # conf / task / user / p2p / charts / home
│   ├── src/components/ store/ router/ request/ common/
│   └── vite.config.js            # 构建产物输出到 ../src/static，并刷新 index.tpl
├── Dockerfile                    # 三阶段构建（golang / node / alpine）
├── control                       # build / start / stop / init / pack 等脚本
├── gopub.service                 # systemd unit
├── gopub-kubernetes.yml          # Deployment + Service（含 MySQL sidecar 示例）
└── .github/workflows/
    ├── ci.yml                    # push / PR 跑构建、vet、测试
    └── release.yml               # 打 v* tag 自动打包发 Release
```

## 发布流程详解

服务端收到 `/api/get/walle/release` 后异步执行（`src/controllers/walle/release.go:108`），每一步都会写 `record` 表并推进进度码：

| 进度码 | 步骤 | 说明 |
| --- | --- | --- |
| 10 | `InitLocalWorkspace` / `InitRemoteVersion` | 生成版本号 `YYYYMMDD-HHMMSS`，创建本地工作空间与目标机版本目录 |
| 20 | `PreDeploy` | 发布机本地前置钩子 |
| 30 | `UpdateToVersion` | 按 `repo_type` 从 Git / 文件地址 / Jenkins 取代码到指定版本 |
| 40 | `PostDeploy` | 发布机本地后置钩子（编译、依赖安装通常放这里） |
| 50 | `CopyFiles` | 打包（可 gzip、按 `excludes` 排除）后通过 SFTP 或 P2P 分发到全部目标机 |
| 60 | `UpdateRemoteServers` | 目标机上依次执行 `pre_release` → 软链切换/移动目录 → `post_release` |
| — | `LastDeploy` | 全部机器完成后，在发布机本地执行统一收尾命令 |
| 100 | `CleanUpLocal` / `CleanUpReleasesVersion` | 清理本地工作空间，按 `keep_version_num` 清理历史版本 |

工作空间路径规则：`{deploy_from}/{env}/{project}-{版本号}`，其中 `env` 由项目 `level` 决定（`1`=test、`2`=simu、`3`=prod）。

**回滚**：上线单 `action != 0` 时走 `rollBackHandling`，直接对目标机执行 `UpdateRemoteServers`（把软链指回选定的历史版本），不重新拉代码，因此回滚是秒级的。前提是该版本仍在 `keep_version_num` 保留范围内。

**上线单状态**（`task.status`，文案见 `src/controllers/task/list.go:56`）：

| 值 | 含义 |
| --- | --- |
| `0` / `1` | 新建提交 |
| `2` | 审核拒绝 |
| `3` | 上线完成 |
| `4` | 上线失败 |

`is_run=1` 时列表页显示为「上线中」。

**并发保护**：上线单 `is_run=1` 期间拒绝重复触发；`status=2`（审核拒绝）或 `3`（上线完成）的单子不允许再次发布；非项目所有者且非管理员不能发布他人的单子（`release.go:38`）。

## 环境要求

| 组件 | 版本 | 说明 |
| --- | --- | --- |
| Go | >= 1.25 | 见 `go.mod` |
| Node.js | >= 20.19.0 | `frontend/package.json` 的 `engines` |
| npm | >= 10 | 同上 |
| MySQL | 5.7+ 或兼容 | 首次启动会自动 `CREATE DATABASE IF NOT EXISTS` 并建表 |
| Git | 任意 | 发布机需要能 clone 业务仓库 |
| SSH | OpenSSH | 发布机需能免密登录目标机 |

构建阶段需要能访问 Go module proxy 与 npm registry。

## 快速开始

### 1. 准备 SSH 免密

gopub 以「发布机」身份用 SSH 登录目标机，所以运行 gopub 的系统用户需要有密钥对，且公钥要加入目标机**发布用户**（项目里的 `release_user`）的 `~/.ssh/authorized_keys`：

```shell
ssh-keygen -t rsa -b 4096 -N '' -f ~/.ssh/id_rsa
ssh-copy-id -i ~/.ssh/id_rsa.pub deploy@目标机IP
```

### 2. 配置数据库

仓库里只有模板 `src/conf/app.conf.example`，实际使用的 `src/conf/app.conf` 不入库（见 `.gitignore`），需要从模板复制一份——`./control start|run|rundocker|init` 检测到它缺失时也会自动复制：

```shell
cp src/conf/app.conf.example src/conf/app.conf
```

然后编辑 `src/conf/app.conf`，按运行模式（`[dev]` / `[prod]` / `[docker]`）填写数据库连接：

```ini
runmode = prod          # 决定读取哪个段的配置

[prod]
HttpPort  = 8192
mysqluser = "root"
mysqlpass = "你的密码"
mysqlhost = "127.0.0.1"
mysqlport = 3306
mysqldb   = "go_pub"
SecretKey = "换成足够随机的长字符串"
```

> 配置文件只会从**运行目录下的 `conf/app.conf`** 或可执行文件同级的 `conf/app.conf` 加载（`src/library/config/config.go:43`），没有 `app.local.conf` 之类的覆盖机制。改配置请直接改 `src/conf/app.conf`；该文件已从版本控制中移出，不会被 `git pull` 或升级解压覆盖，也不会被误提交。新增配置项时记得同步补进 `app.conf.example`，否则新部署拿不到它。

### 3. 构建前端

```shell
cd frontend
npm ci
npm run build        # 产物写入 ../src/static，并自动刷新 ../src/views/index.tpl
```

前端热开发（默认把 `/api` 代理到 `127.0.0.1:8192`）：

```shell
cd frontend && npm run dev
```

构建产出零警告。两处需要留意的约定：

- `frontend/vite.config.js` 里关掉了 Rolldown 的 `checks.invalidAnnotation`。原因是 `@vueuse/core`（Element Plus 的传递依赖）产物中有位置不被 Rolldown 识别的 `/* #__PURE__ */` 注释，第三方源码改不了，且只影响它自身的 DCE 粒度。其余诊断照常输出，业务代码写出无效注解仍需自查。
- 只被 `router/index.js` 动态导入的组件**不要**再放进 `frontend/src/components/index.js` 这个桶文件。桶文件被 `App.vue` 静态引用，一旦写进去该组件就会被并入首屏 chunk，动态导入失效。`leftSlide` 原先就踩了这个坑，现已从桶里移出、独立成约 3.5 kB 的 chunk；同类的 `leftSlideTologin` 一直只走动态导入，不要补进桶文件。

### 4. 构建并启动后端

```shell
./control build      # gofmt + go build，产物为 src/gopub
./control init       # 建库建表 + 写入初始管理员（等价于 src/gopub -syncdb）
./control start      # 后台启动，pid 写入 gopub.pid
./control status     # 查看状态
./control tail       # 跟踪 src/logs/stdout.log
./control stop       # 停止
```

访问 `http://127.0.0.1:8192`，默认管理员账号 `admin`，初始密码见 `src/models/AdminInit.go` 中写入的 bcrypt 哈希（沿用上游默认值，**首次登录后请立即修改**）。

`control` 脚本可用子命令：`build | pack | start | stop | kill | restart | reload | status | run | rundocker | init | tail | sslkey`。

> **升级不会覆盖配置**：发布包（`control pack` 与 Release 附件）里只有 `src/conf/app.conf.example`，**不含 `app.conf`**，所以解压到现有目录只会换掉二进制与前端静态资源，你填好的数据库密码、`SecretKey` 等原样保留。首次部署时 `./control start` 会自动从模板生成一份。若你从更早的版本升级，目标机上那份 `app.conf` 本来就在，不需要任何额外操作。

> **从旧版本升级**：`user` 表新增了 `auth_key_expire_at` 列（登录凭据过期时刻）。`./control init` 会建它，但通常升级时只换二进制、不重跑 init，所以启动时也会检查一次并自动补列（`src/models/user_authkey.go` 的 `MigrateAuthKeyColumn`）。若数据库账号没有 DDL 权限，补列会失败并在日志里明确提示——**这种情况下登录不可用**，需要手动执行 `./control init` 或补一句 `ALTER TABLE user ADD COLUMN auth_key_expire_at datetime NULL;`。升级后所有人需重新登录一次，详见[内部接口](#3-内部接口)。

## 配置说明

配置文件为 ini 格式，键名大小写不敏感（内部统一转小写），取值时**先查当前 `runmode` 段，取不到再回落全局段**。

### 服务

| 键 | 默认值 | 说明 |
| --- | --- | --- |
| `appname` | `gopub` | 应用名 |
| `runmode` | `prod` | `dev` / `prod` / `docker`，决定生效的配置段 |
| `HttpAddr` | `0.0.0.0` | 监听地址 |
| `HttpPort` | `8192` | 监听端口（全局段的 `httpport = 8080` 会被各模式段覆盖） |
| `EnableGzip` | `true` | 响应 gzip |
| `AccessLogs` | 按段 | 是否打印访问日志 |
| `Graceful` | `false` | 为 `true` 时收到退出信号会等待在途请求（10s 超时） |
| `SshPort` | `22` | 连接目标机的 SSH 端口 |
| `SecretKey` | 空 | **JWT 签名密钥，必须填写足够随机的长字符串** |
| `authKeyLifetime` | `604800` | 控制台登录凭据有效期（秒），滑动过期。配成非正数时回落到默认的 7 天 |
| `SessionOn` / `SessionGCMaxLifetime` / `SessionCookieLifeTime` | `true` / `86400` / `86400` | beego 遗留项，迁移到 Echo 后已无代码读取，改动无效果 |
| `AutoRender` / `CopyRequestBody` / `EnableDocs` / `EnableHTTP` / `HttpsPort` / `EnableAdmin` / `AdminAddr` / `AdminPort` | — | 同为 beego 遗留项；请求体重复读取现已由 `src/controllers/base.go` 无条件支持 |

### 数据库

| 键 | 说明 |
| --- | --- |
| `mysqluser` / `mysqlpass` / `mysqlhost` / `mysqlport` / `mysqldb` | 连接参数 |
| `db_max_idle_conn` / `db_max_open_conn` | 连接池的常驻空闲连接数与最大连接数（默认 30 / 100） |
| `db_conn_max_lifetime` | 单条连接最长存活秒数（默认 3600），须小于 MySQL 的 `wait_timeout` 与中间层空闲超时 |
| `db_conn_max_idle_time` | 空闲连接最长保留秒数（默认 600），闲时据此收缩连接池 |

连接全程复用：进程启动时建立一个全局连接池（`src/library/db/db.go` 的 `Init`），查询从池中借出、用完归还，不存在一次请求一次 TCP + 认证握手。空闲连接常驻 `db_max_idle_conn` 条，超过 `db_conn_max_idle_time` 未使用才回收；任一连接活过 `db_conn_max_lifetime` 会被主动淘汰重建，避免用到被 MySQL 或中间层单方面关闭的死连接。`db.Stats()` 可读取池的实时状态（在用/空闲/等待次数）用于排查连接不足。

`runmode=docker` 时，环境变量 `MYSQL_USER` / `MYSQL_PASS` / `MYSQL_HOST` / `MYSQL_PORT` / `MYSQL_DB` 优先于配置文件（`src/library/db/db.go:91`）。**注意：非 docker 模式下这些环境变量不生效。**

### Jenkins / 邮件 / P2P

| 键 | 说明 |
| --- | --- |
| `JenkinsUserName` / `JenkinsPwd` | Jenkins 凭据，同名环境变量优先（`src/main.go:54`） |
| `emailUsername` / `emailPwd` / `emailHost` / `emailPort` | 通知邮箱 |
| `AgentDir` / `AgentDestDir` | P2P agent 本地目录与下发到目标机的目录 |

### LDAP

| 键 | 说明 |
| --- | --- |
| `enableLdap` | 为 `true` 时改用 LDAP 认证，`false` 用本地数据库账号 |
| `ldapHost` / `ldapPort` | LDAP 服务器 |
| `ldapPeopleDn` / `ldapPeopleDnTpl` | 用户 DN 与登录模板，支持 `{uid}` 占位 |
| `ldapGroupDn` / `ldapGroupFilter` | 用户组查询，支持 `{UidNumber}` `{uid}` `{cn}` `{sn}` 占位 |
| `ldapGroupName2roleid_gopubAdmin` / `_gopubPre` / `_gopubSingle` | LDAP 组到角色（1 / 10 / 20）的映射，需在 LDAP 中建对应三个组 |

### JumpServer

| 键 | 说明 |
| --- | --- |
| `enableJumpserver` | 开启后可在项目配置里选服务器分组，发布时实时拉取分组 IP |
| `jumpserver` / `jump_username` / `jump_password` | 地址与凭据 |
| `jump_auth_api` / `jump_grouplist_api` / `jump_groupid2ips_api` | API 路径（默认适配 JumpServer 1.5.3） |

## Docker 部署

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

镜像特点：

- 三阶段构建 —— `golang:1.25-alpine` 编译后端、`node:22-alpine` 构建前端、`alpine:3.22` 作为运行时；
- 运行时镜像内置 `bash git openssh curl wget tzdata`，时区固定 `Asia/Shanghai`；
- 构建时会执行 `ssh-keygen` 生成 `/root/.ssh/id_rsa` 并把公钥打印到构建日志，**需要把这段公钥加入目标机的 `authorized_keys`**；
- 入口是 `./control rundocker`，即以 `-docker` 参数启动，会自动建库建表并让 `MYSQL_*` 环境变量生效。

> 生产环境建议把 `/root/.ssh` 与 `src/conf`、`src/logs` 挂载为卷，否则重建容器会丢失密钥、配置与日志。镜像里只打包 `src/conf/app.conf.example`，入口 `./control rundocker` 会在 `app.conf` 缺失时从模板生成，所以挂空卷首次启动也能跑起来（数据库连接由 `MYSQL_*` 环境变量注入）。

## Kubernetes 部署

```shell
kubectl apply -f gopub-kubernetes.yml
```

`gopub-kubernetes.yml` 提供的是**示例**清单，包含一个 Deployment（gopub + MySQL 5.7 sidecar）与一个 NodePort Service（`8192`）。直接上生产前至少要改这几处：

- `image` 换成你自己构建推送的镜像；
- 密码从 `env` 明文改为 `Secret`；
- MySQL 从 sidecar + `hostPath` 改为独立的有状态服务或云数据库；
- `securityContext.privileged: true` 按需收紧；
- `hostAliases` 里的示例域名/IP 换成你的 GitLab、Jenkins 地址。

## systemd 托管

```shell
# 假设部署在 /www/server/gopub（gopub.service 里的默认路径）
cp gopub.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now gopub
systemctl status gopub
```

`gopub.service` 是 `Type=forking`，直接复用 `control start|restart|stop`。若部署路径不同，记得改 unit 里的三条 `Exec*` 路径。

## 发布制品与 CI

本地打包：

```shell
./control pack       # 生成 ../gopub.tar.gz，含 control、service、二进制、conf、views、static、agent
```

日常校验（`.github/workflows/ci.yml`）：推分支和提 PR 都会触发，后端跑 `gofmt -l src/` 检查、`go build ./...`、`go vet ./...`、`go test ./...`，前端跑 `npm ci` + `npm run build`。其中 `src/routers/router_test.go` 是重点——新增路由若忘了挂 `RequireLogin`，会在这一步失败。

CI 打包（`.github/workflows/release.yml`）：推送形如 `v1.2.3` 的 tag 时自动触发，流程为「构建前端 → `go build -trimpath -ldflags "-s -w"` 构建后端 → 按 `control pack` 的清单打成 `gopub-<tag>.tar.gz` → 创建 GitHub Release 并上传附件」，Release Notes 自动生成。

```shell
git tag v1.2.3 && git push origin v1.2.3
```

## 数据模型

`./control init`（或首次以 `-docker` 启动）会自动建库并 `AutoMigrate` 以下表：

| 表 | 说明 |
| --- | --- |
| `user` | 用户，含角色 `role`、`from_ldap` 标记、bcrypt 密码哈希 |
| `group` | 用户与项目的授权关系（`project_id` + `user_id` + `type`） |
| `project` | 项目配置：仓库、目标机、发布目录、发布方式、钩子、保留版本数、P2P/gzip/SSH 算法开关等 |
| `task` | 上线单：分支、commit、文件列表、状态、进度、是否可回滚、执行中标记 |
| `record` | 每条命令的执行记录（命令、耗时、返回、进度码），页面进度条数据来源 |
| `task_err_log` | 发布失败的错误信息 |
| `session` | beego 时代的会话表，仍会被建出来，但当前代码没有任何地方读写它 |
| `api_system` | 对外 API 的 appid / appsecret / IP 白名单 |
| `migration` | 迁移记录 |

## 对外 API

`/v1` 下的接口面向外部系统，使用 JWT（HS256，密钥为配置里的 `SecretKey`）鉴权。

### 1. 换取 token

```http
GET /v1/token?appid=<appid>&appsecret=<appsecret>
```

服务端会校验三件事：appid 存在、appsecret 匹配、**请求方 IP 在该 appid 的白名单内**（`api_system.ip`，逗号分隔）。

成功：

```json
{ "access_token": "eyJhbGciOiJIUzI1NiIs...", "expires_in": "1750000000" }
```

失败：

```json
{ "errcode": "100", "errmsg": "appid不存在 " }
```

token 有效期 1 小时，`iss` 为 appid，`expires_in` 是过期时刻的 Unix 时间戳（不是剩余秒数）。

> `/v1` 的凭据是**系统级**的：token 只绑定 appid，不关联任何 gopub 用户，因此**不受[对象级权限](#对象级权限)约束**，持有者可读写全部项目与上线单。这是它作为外部集成接口的设计，把关的是 appsecret 与 IP 白名单——请按系统级凭据妥善保管，不要发给个人使用。

### 2. 上线单接口

后续请求把 token 放在 `Authorization` 请求头里（**直接放裸 token，不带 `Bearer ` 前缀**）：

| 方法与路径 | 说明 |
| --- | --- |
| `POST /v1/task` | 创建上线单 |
| `GET /v1/task` | 按项目名与时间区间查询上线单列表 |
| `GET /v1/task/:id` | 查询单个上线单 |
| `PUT /v1/task/:id` | 更新上线单 |
| `DELETE /v1/task/:id` | 删除上线单 |

```shell
TOKEN=$(curl -s "http://127.0.0.1:8192/v1/token?appid=1&appsecret=xxx" | jq -r .access_token)
curl -H "Authorization: $TOKEN" "http://127.0.0.1:8192/v1/task/1"
```

错误码：`102` 缺少 token，`103` token 校验失败（过期、签名不符、算法不是 HS256）。

### 3. 内部接口

Web 控制台使用的是 `/api/get/*` 与 `/api/post/*` 系列（项目配置、上线单、Git/Jenkins 查询、发布执行、P2P、发布记录、用户），接口清单见 `src/routers/router.go`。

它们的鉴权与 `/v1` 不同：登录成功后服务端签发一个 32 位随机串（`crypto/rand`）存入 `user.auth_key` 并返回，前端后续请求带 `Authorization: TOKEN <auth_key>` 头，服务端反查 `user` 表识别身份（`src/controllers/auth.go:78` 的 `userByToken`）。

凭据生命周期由 `src/models/user_authkey.go` 统一管理：

- **有效期**：由 `authKeyLifetime` 配置（秒，默认 `604800` 即 7 天），过期时刻存在 `user.auth_key_expire_at`；
- **滑动续期**：有请求就顺延，闲置超过 `authKeyLifetime` 后失效。为避免每个请求都写库，只在剩余有效期不足一半时才 `UPDATE`；
- **多端共存**：重新登录时若已有未过期的 `auth_key` 就沿用原值、只顺延过期时间，因此在多台机器上登录不会互相踢掉；
- **吊销**：`POST /logout` 与改密码会把 `auth_key` 清空、`auth_key_expire_at` 置 NULL，该账号所有端立即失效。

`auth_key_expire_at` 为 NULL 的记录一律视为未登录，所以升级到本版本后所有人都需要重新登录一次——这正是为了吊销升级前签发的那批无过期时间的凭据。

#### 页面级权限（管理员门禁）

前端菜单里标了 `adminOnly` 的那些页面——项目配置、全部上线单、运维工具、用户管理——只对 `role=1` 开放。这道门禁在前后端各挂一次：

- 前端：路由 `meta.admin` + `frontend/src/router/index.js` 的 `beforeEach` 守卫。光靠菜单过滤挡不住直接在地址栏敲 URL。
- 后端：`src/routers/router.go` 里用 `adminGET` / `adminPOST` 注册，中间件是 `src/controllers/auth.go` 的 `RequireAdmin`。判断依据是「该接口的全部调用方都是 admin 页面」，被普通用户页面用到的接口（`conf/get`、`conf/mylist`、`task/mylist` 等）仍只要求登录。

`src/routers/router_test.go` 的 `wantAdminRoutes` 锁定了这份清单，`src/controllers/auth_test.go` 直接覆盖中间件本身对各角色的放行 / 拒绝。新增管理员接口时两处都要同步。

> **本版本移除了免登录访问**。此前 `/api/get/task/list`、`/api/get/task/get`、`/api/get/conf/get`、`/api/get/record/list`、`/api/get/record/attempts` 五个接口在免登录白名单里，支撑 `searchtaskList` / `searchtaskRelease` 两个匿名查询页。那两个页面与 `taskList` / `taskRelease` 功能重合，同时也是绕过上面 admin 门禁的口子，已连同白名单、登录页的「上线单查询」入口一并删除。现在除 `/login`、`/loginbydocke`、`/`、`/v1/token` 外，所有接口都要求登录。

#### 对象级权限

登录之外还有一层**对象级权限**：能不能读写某个项目 / 上线单由 `src/controllers/perm.go` 统一判定，语义与项目列表的过滤规则（`src/controllers/conf/mylist.go:30-35`）保持一致——**列表里看得到的，才能操作**：

| `user.role` | 含义 | 可操作的项目 |
| --- | --- | --- |
| `1` | 管理员 | 全部 |
| `10` | 全部预发布用户 | 仅 `level=2`（simu） |
| `20` | 单个项目用户 | 仅 `group` 表里授权过的 |
| 其他（含 `0`） | 未知角色 | 无 |

落地方式是在上下文加载处统一收口：`ctx.Project` / `ctx.Task` 只有在当前用户确实有权限时才会被挂上（`src/controllers/base.go:55-64`），否则保持 `nil`，由各 handler 已有的空值判断拦下。不走上下文、自己按 ID 查库的写接口（改项目配置、改上线单、删除、锁定、批量刷新等）则逐个显式调用 `CanAccessProject` / `CanAccessTask`。判断一律以**库里的旧记录**为准，不采信请求体里的 `level`、`project_id` 等字段——否则伪造一下就绕过去了。

> ⚠️ **升级注意**：如果你用 LDAP 登录且某些组没有在 `app.conf` 里配 `ldapGroupName2roleid_*`，这些用户的 `role` 会落到 `0`。收紧之前他们在列表里能看到**全部项目**，收紧之后将**完全没有项目权限**。升级前请先执行 `SELECT id, username, role FROM user WHERE role NOT IN (1, 10, 20);` 确认没有这类账号，或补上对应的角色映射。

## P2P 分发

大包多机分发时，逐台 SFTP 推送会把发布机出口带宽打满。gopub 内置了一套 BitTorrent 实现来解决这个问题：

- **Server** 跑在 gopub 进程内（非 docker 模式下由 `src/main.go:126` 的 `init_sever.Start()` 启动），负责做种、建任务、查任务；
- **Agent** 是下发到目标机的独立二进制（`src/agent/`），互相交换分片；
- 默认端口：管理 `45003`、数据 `45002`，agent 侧同样有管理/数据端口；支持限速（MBps）、内存缓存大小、并发任务数配置，详见 `src/agent/README.md` 与 `src/agent/server.json`；
- Web 上有「agent 状态查询」页，`src/tasks/check_p2p_agent.go` 会周期性巡检 agent 存活；
- 在项目配置里打开 `p2p` 开关后，`CopyFiles` 阶段就会走 P2P 通道而不是 SFTP。

## 安全须知

⚠️ 这套系统持有目标机的 SSH 密钥并能在上面执行任意命令，等同于一台跳板机，请按高敏感系统对待。

- **必须修改 `SecretKey`**：它是 JWT 的签名密钥，模板里留空，从模板生成配置后务必填一个足够随机的长字符串。
- **必须修改默认管理员密码**：`admin` 的初始哈希是上游硬编码的公开值。（上游还预置了一个固定的 `auth_key`，等于把一个公开的 admin 凭据装进每套部署，现已改为留空、由首次登录签发。若你的库是早先建的，请确认 `user` 表里没有残留 `cJIrTa_b2Hnjn6BZkrL8PJkYto2Ael3O` 这个值。）
- **不要提交真实凭据**：`src/conf/app.conf` 已从版本控制中移出（`.gitignore`），仓库里只留占位值的 `app.conf.example`。注意**历史提交里仍能查到早先写入的数据库密码**，如果那套凭据还在用，请另行更换。
- **`repo_password` 等字段在数据库中是明文存储**，请限制数据库访问权限。
- **钩子脚本与发布命令会在目标机上以 `release_user` 身份执行**，等于把命令执行权交给了有项目权限的用户。系统内**没有可用的审批环节**（见[已知限制](#已知限制)），所以 `group` 表里的项目授权就是唯一的闸门，务必克制。
- **CORS 当前是 `AllowOrigins: ["*"]`**（`src/routers/router.go:27`），对外暴露前建议收紧到你自己的域名。
- **生产建议走 HTTPS**：`src/controllers/api/base_api.go` 里有一段强制 HTTPS 的检查目前是注释状态，按需打开，或在前面挂反向代理。
- 目标机的 `release_user` 建议只给必要目录的写权限，不要直接用 root。

## 已知限制

- 测试覆盖仍然偏低：只有少数几个测试文件——`src/library/ssh/remote_test.go`（SSH 算法档位）、`src/routers/router_test.go`（路由清单、免登录拦截与管理员路由清单）、`src/controllers/auth_test.go`（`RequireLogin` / `RequireAdmin` 对各角色的放行与拒绝）、`src/library/common/shell_test.go`（shell 转义与 git 引用名校验）、`src/controllers/perm_test.go`（对象级权限矩阵），其余包大多是 `no test files`；前端没有测试。
- 有四个「配了也不生效」的项目字段：`post_release_together`（"所有服务器部署完成后统一执行"）、`gzip`、`audit`（审核开关）、`view_history`。它们在 `models.Project` 里有定义，前端表单里也有默认值，但后端没有任何地方读取，配置了不会产生行为差异。需要收尾命令请用 `last_deploy`；差异对比请用 `/api/get/task/changes`（它不看 `view_history`）。（注意 `app.conf` 里的 `EnableGzip` 是另一回事，那是 HTTP 响应压缩，正常生效。）
- **没有发布审批环节**：`status=2`（审核拒绝）这个状态存在，但代码里没有任何地方会把上线单置为 2，也没有审批接口——有项目权限的人可以直接发布。
- 登录用户之间还有少量横向可见：`/api/get/record/attempts` 只按 `taskId` 查，不校验调用者对该上线单所属项目有没有权限。它只返回批次号、时间与成败，不含命令与主机信息，但严格说仍是越权可读。
- Element Plus 是 `app.use(ElementPlus)` 全量注册，构建后单独成 `element-plus` chunk（约 970 kB / gzip 311 kB）且属首屏依赖。改为按需引入可显著瘦身，但需要同时接管代码里 54 处 `this.$message` / `this.$confirm` 这类全局属性。
- JumpServer 对接按 1.5.3 版本 API 编写，新版 JumpServer 需要自行适配。

## 常见问题

**Q：启动报「找不到配置文件 conf/app.conf」？**
两种可能。一是还没从模板生成配置：`cp src/conf/app.conf.example src/conf/app.conf`（`./control start` 会自动做这一步）。二是工作目录不对——配置是相对**工作目录**查找的，请通过 `./control start`（内部会 `cd src/`）启动，或手动进入 `src/` 后再运行二进制。

**Q：端口到底是 8080 还是 8192？**
全局段写的是 `httpport = 8080`，但 `[dev]`/`[prod]`/`[docker]` 三个段都写了 `HttpPort = 8192`，而分段配置优先，所以实际是 **8192**。

**Q：设置了 `MYSQL_HOST` 但没生效？**
`MYSQL_*` 环境变量只在 `runmode=docker` 时才覆盖配置文件。非容器部署请直接改 `app.conf`。

**Q：发布卡在 50（分发文件）？**
先用项目页的「检测项目」功能（`/api/get/walle/detectionssh`）验证 SSH 免密与目标目录权限；老旧 sshd 请把项目的 SSH 算法档位切到 `legacy`。

**Q：回滚选不到想要的版本？**
历史版本受 `keep_version_num` 限制，超出份数的目录在发布结束时已被清理。

## 开发约定

- Go 代码统一用 `gofmt`（`./control build` 会先跑 `gofmt -w src/`）；包名短、小写、与目录同名。
- 控制器按业务分目录（`task` / `conf` / `api` / `walle` / `user` / `p2p` / `record` / `other`），新处理器放到相关目录下。
- 依赖用 Go modules 管理，**不要重新引入 `vendor/`**。
- 前端 Vue 3 + Vuex 4 + Vue Router 4 + Element Plus；store 模块放 `frontend/src/store/`，路由放 `frontend/src/router/`，API 地址常量放 `frontend/src/common/port_uri/`；优先用 ESM。
- 新增 Go 测试放在被测包旁边命名为 `*_test.go`，优先表驱动；前端测试用 `*.spec.js` / `*.test.js` 并在 `package.json` 里登记脚本。
- 提交信息简短、用动词开头；PR 请说明行为变化、列出手工验证命令、标注配置/数据库影响，UI 改动附截图；不要把依赖升级和功能改动混在一个提交里。

验证命令：

```shell
go build ./...
go vet ./...
go test ./...
cd frontend && npm run build
cd frontend && npm audit --audit-level=high
```

## 致谢与许可

本仓库是 [linclin/gopub](https://github.com/linclin/gopub) 的衍生作品，在其基础上做了 Beego → Echo 迁移、JWT 库更换、SSH 算法档位、Release 打包流水线等改动，完整改动清单见 [NOTICE](NOTICE) 与 git 历史。

发布协议：[Apache License 2.0](LICENSE)。原始版权归 linclin 及其贡献者所有（Copyright 2018 linclin），衍生部分 Copyright 2026 lnatpunblhna。
