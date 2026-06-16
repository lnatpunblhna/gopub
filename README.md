# gopub

gopub 是一个面向运维发布场景的 Web 发布系统，支持项目配置、上线单创建、Git/Jenkins 发布、SSH 执行、文件同步、回滚和发布记录查询。当前代码已经升级为 Go modules 后端和 Vue 3 前端，适合作为二次开发基础。

## 技术栈

- 后端：Go 1.25、Beego 1.x、MySQL、httprouter
- 前端：Vue 3、Vue Router 4、Vuex 4、Vite 8、Element Plus、Axios、Sass
- 部署：Docker、多阶段构建、systemd、Kubernetes manifest

## 项目结构

```text
.
├── src/                 # Go 后端、模板、静态资源
│   ├── controllers/     # Web/API 控制器
│   ├── library/         # 发布、SSH、P2P、LDAP 等业务库
│   ├── models/          # ORM 模型和初始化数据
│   ├── routers/         # Beego 路由注册
│   ├── conf/app.conf    # 本地配置
│   ├── static/          # Vite 构建产物
│   └── views/index.tpl  # Beego 入口模板
├── frontend/            # Vue 3 + Vite 前端
├── Dockerfile
├── control              # 构建/启动/初始化脚本
└── gopub-kubernetes.yml
```

## 环境要求

- Go >= 1.25
- Node.js >= 20.19.0，npm >= 10
- MySQL 5.7+ 或兼容版本
- 构建时需要访问 Go module proxy 和 npm registry

## 本地开发

```shell
# 安装前端依赖
cd frontend
npm ci

# 启动前端开发服务器，默认代理 /api 到 127.0.0.1:8192
npm run dev

# 构建前端，产物写入 ../src/static，并刷新 ../src/views/index.tpl
npm run build
```

```shell
# 返回仓库根目录，配置数据库
cp src/conf/app.conf src/conf/app.local.conf

# 编译后端
./control build

# 初始化数据库
./control init

# 启动服务
./control start
```

默认访问地址为 `http://127.0.0.1:8192`。初始化账号通常为 `admin / 123456`，以实际数据库初始化逻辑为准。

## 验证命令

```shell
go test ./...
cd frontend && npm run build
cd frontend && npm audit --audit-level=high
```

`npm run build` 当前会输出 Sass `@import` 和大 chunk 警告；这些不阻断构建，后续可通过迁移 Sass module 语法和拆分 Element Plus 相关 chunk 继续优化。

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
  -d gopub
```

也可以使用 `gopub-kubernetes.yml` 部署到 Kubernetes：

```shell
kubectl apply -f gopub-kubernetes.yml
```

## 配置与安全

- 数据库、Jenkins、LDAP、SSH 等配置位于 `src/conf/app.conf`。
- Docker 运行时优先使用 `MYSQL_HOST`、`MYSQL_PORT`、`MYSQL_USER`、`MYSQL_PASS`、`MYSQL_DB` 等环境变量。
- 不要提交真实密码、私钥、生产数据库地址或 Jenkins token。
- gopub 运行用户的 SSH public key 需要加入目标机器发布用户的 `~/.ssh/authorized_keys`。

## 发布流程概览

1. 在项目配置中维护仓库地址、目标机器、发布目录、保留版本数和钩子脚本。
2. 创建上线单，选择 Git 分支、tag、commit 或 Jenkins build。
3. 执行发布，系统完成拉取、打包、同步、软链切换和日志记录。
4. 如发布失败或需要恢复，可基于保留版本创建回滚任务。

## 维护说明

- Go 依赖使用 modules 管理，不再提交 `vendor/`。
- 前端入口是 `frontend/index.html` 和 `frontend/src/main.js`。
- 前端 API 地址常量在 `frontend/src/common/port_uri/`。
- Beego 根路由渲染 `src/views/index.tpl`，该文件由 Vite 构建自动刷新。
