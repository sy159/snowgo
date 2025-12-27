# snowgo <img src="https://img.shields.io/badge/golang-1.24-blue"/> <img src="https://img.shields.io/badge/gin-1.11.0-green"/> <img src="https://img.shields.io/badge/gorm-1.31.1-red"/>
基于Gin + GORM的高可用、模块化 Go Web脚手架，集成常用中间件与企业级基础设施（日志、配置、鉴权、消息队列、分布式锁、codegen、Docker/Compose 支持等），旨在快速搭建中小型项目。

------------
### 🔌 集成组件:
| 🧩 模块        | 🔧 组件                 | 📝 描述                                 |
|--------------|-----------------------|---------------------------------------|
| 🌐 Web 框架      | Gin                   | 高性能 HTTP 框架                           |
| ⚙️ 配置管理     | Viper                 | 灵活的配置加载支持                             |
| 📜 日志系统     | Zap + ELK             | 支持多格式输出，可集成 ELK 进行日志分析(对敏感数据进行脱敏处理)   |
| 🗃️ 数据访问     | GORM + Gen            | ORM 工具，支持读写分离、多数据库配置                  |
| 🚀 缓存系统     | go-redis              | 高性能 Redis 客户端封装                       |
| 🔐 鉴权系统     | JWT                   | 支持 access_token / refresh_token 的鉴权方案 |
| 🛂 权限系统     | 自定义 RBAC（菜单树）         | 支持按钮/接口权限，基于菜单树结构                     |
| 🛡️ 限流中间件   | 自研 Rate Limiter       | 支持 IP / 路由维度的限流控制                     |
| 🔗 中间件       | 跨域、日志、异常处理            | 全面覆盖 Web 常用中间件                        |
| 🧵 分布式能力   | Redis Lock、RabbitmqMQ | 实现分布式锁、事件驱动架构                         |
| 📈 可观测性     | Prometheus + Grafana  | 实现服务监控指标管理等                           |


------------
### 🚀 快速开始
#### 环境准备
- Go >= 1.24
- Docker & Docker Compose（若使用容器）
- GNU Make
#### 项目拉取
```shell
git clone https://github.com/sy159/snowgo.git
cd snowgo
```
------------

#### 1. 修改配置
修改配置文件
```shell
# 推荐在本地开发使用 ENV=dev，容器环境使用 ENV=container 对应 config.container.yaml
vim config$.{env}.yaml
```
#### 2. 初始化(可选)
```shell
make mysql-init # 如果包含初始化脚本
make mq-init    # RabbitMQ/Pulsar 声明
```

------------
#### 3. 运行项目
![](/assets/images/run.png)
##### 3.1 💻 本地运行
安装运行需要的依赖
```shell
go mod download
go mod tidy
```
直接运行（适合开发调试）
```shell
go run ./cmd/http  # http服务
go run ./cmd/consumer  # mq消费服务(根据需求可选)
```

------------
##### 3.2 🐳 Docker 运行
构建镜像
```shell
# API 镜像
make api-build
# 或手动
docker build -t snowgo:1.0.0 .

# Consumer 镜像
make consumer-build
docker build -f Dockerfile.consumer -t snowgo-consumer:1.0.0 .

```
运行单个容器
```shell
# API
make api-run
# 或手动
docker run -d \
  --restart unless-stopped \
  --name snowgo-service \
  -p 8000:8000
  -e ENV=dev \
  -v ./config:/app/config \
  -v ./logs:/app/logs \
  snowgo:1.0.0

# Consumer
make consumer-run
# 或手动
docker run -d \
  --restart unless-stopped \
  --name snowgo-consumer-service \
  -e ENV=dev \
  -v ./config:/app/config \
  -v ./logs:/app/logs \
  snowgo-consumer:1.0.0
```

------------
##### 3.3 🛠 Docker Compose 部署
生成项目服务docker镜像
```shell
# API 镜像
make api-build
# 或手动
docker build -t snowgo:1.0.0 .
```
配置.env相关信息(服务端口、使用镜像等)
```shell
vim .env  # 修改ENV=container，会使用config.container.yaml的配置文件，里面包含了数据库、redis、nginx
```
修改配置文件(地址更换完docker compose服务名)
```shell
vim config$.{env}.yaml
```
启动项目
```shell
# 启动 mysql/redis/nginx 等（由 docker-compose.yml 定义）
make up
# 停止并清理
make down
```


------------
### 🧬 项目结构
```
snowgo
├── .github                 # github cicd
├── assets                  # 静态文件
├── cmd                     # 项目启动入口
├── config                  # 配置文件
├── depoly                  # 部署示例：elk / monitor / rabbitmq 等
├── docs                    # 放置swagger，db.sql等文档
├── internal                # 应用实现（api, dal, di, router, service, worker, server）
│   ├── api
│   ├── constant            # 应用常量
│   ├── dao                 # 数据处理层
│   ├── di                  # 依赖管理
│   ├── router              # web路由&&中间件
│   ├── dal                 # 数据库model query定义
│   │   ├── cmd             # 使用gen生成model跟query、使用init初始化数据
│   │   ├── model           # 生成的model
│   │   ├── query           # model对应的query
│   │   ├── repo            # db的repo
│   │   └── query_model.go  # 需要生成的model列表
│   ├── server              # 服务相关
│   ├── worker              # 后台工作任务
│   └── service             # 业务处理层
├── logs                    # 日志
├── test                    # 测试用例
├── pkg                     # 公共工具库（xlogger, xmq, xdatabase, xauth, xlock, ...）
├── Makefile                # 常用构建/运行脚本
├── Dockerfile              # API 镜像构建
├── Dockerfile.consumser    # Consumer 镜像构建
├── go.mod / go.sum
└── README.md
```


------------
### 🔥 常用
#### 🧩 服务入口说明
| 入口 | 说明 |
|----|----|
| cmd/http | 对外 HTTP API 服务 |
| cmd/consumer | MQ 消费服务（无 HTTP 能力） |
| cmd/mq-declarer | MQ 资源声明工具（只在部署时运行） |

> consumer 与 http 服务应独立部署与扩缩容。

#### 📋 常用命令
- `make api-build`  - 构建 API 镜像
- `make api-run`    - 运行 API 容器
- `make gen init`   - gen: 生成 model 并初始化表
- `make gen add`    - gen: 为新表生成 model && query
- `make gen update` - gen: 更新 model && query

#### 📚 API 文档
[项目接口 文档](https://apifox.com/apidoc/shared-becb3022-d340-491c-bdd7-1f4d4b84620f)


------------
### ✏️ 新业务功能开发流程
#### ✅ 标准流程
```
数据库设计
  ↓
Gen 生成 model / query
  ↓
Repo / Dao 实现
  ↓
Service 编排业务逻辑
  ↓
API 层暴露接口
  ↓
路由 & 权限配置
```
#### 🗃️ 数据库与 Gen 使用规范
> ❗ 禁止手动添加或更改 model / query 文件
```shell
# 新增表
make gen add
# 更新表
make gen update
```


------------
### 📢 注意事项
1. 🧱 数据模型管理
    ```
    # 如果需要定制化某个db下model就修改db的地址配置(默认使用配置的数据库地址)
    vim /internal/dal/cmd/gen.go
   
    # 初始化所有的表
    make gen init
   
    # 新增某些表(根据表名)
    make gen add
   
    # 更新以前生成的model
    make gen update
   
    # 根据model生成所有的query
    make gen query
    ```

------------
2. 📚 文档参考
   - [Gin 官方文档](https://gin-gonic.com/)
   - [GORM 文档](https://gorm.io/zh_CN/docs/)
   - [Gen 工具](https://gorm.io/zh_CN/gen/dao.html)
   - [JWT 文档](https://jwt.io/introduction/)
