# snowgo <img src="https://img.shields.io/badge/golang-1.23-blue"/> <img src="https://img.shields.io/badge/gin-1.10.0-green"/> <img src="https://img.shields.io/badge/gorm-1.25.12-red"/>
基于 Gin 开发的高可用、模块化 Go 脚手架，集成丰富的中间件与企业级基础设施，适用于中小型服务系统快速搭建，支持 Docker & Docker Compose 一键部署。

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
| 🛂 权限系统     | 自定义 RBAC（菜单树） | 支持按钮/接口权限，基于菜单树结构                     |
| 🛡️ 限流中间件   | 自研 Rate Limiter     | 支持 IP / 路由维度的限流控制                     |
| 🔗 中间件       | 跨域、日志、异常处理  | 全面覆盖 Web 常用中间件                        |
| 🧵 分布式能力   | Redis Lock、Pulsar MQ | 实现分布式锁、事件驱动架构                         |
| 📈 可观测性     | Prometheus + Grafana  | 实现服务监控指标管理等                           |

[//]: # (1. gin轻量级Web框架)

[//]: # (2. zap日志管理)

[//]: # (3. viper配置文件解析)

[//]: # (4. response统一结构返回，以及error code自定义)

[//]: # (5. gorm数据库组件，以及使用gen生成model以及query&#40;支持读写分离以及多数据库配置&#41;)

[//]: # (6. go-redis缓存组件)

[//]: # (7. jwt鉴权)

[//]: # (8. rate限流)

[//]: # (9. 访问日志、跨域、全局异常处理等中间件)

[//]: # (10. 基于redis等实现的分布式锁)

[//]: # (11. mq&#40;pulsar&#41;)

[//]: # (12. elk收集日志在kibana展示)

[//]: # (13. Prometheus+Grafana实现监控)

------------
### 🧬 项目结构
```
snowgo
├── .github  github cicd
├── assets  静态文件
├── config  配置文件
├── depoly
│   ├── elk  elk部署
│   └── monitor 监控部署
├── docs  放置swagger，db.sql等文档
├── internal  应用程序
│   ├── api    主要处理用户请求
│   │   ├── account  账户相关业务
│   │   │   └── user.go
│   │   └── api.go
│   ├── constants  应用常量
│   │   └── constants.go
│   ├── dao    数据处理层
│   │   └── dao.go
│   ├── di    依赖管理
│   │   └── container.go
│   ├── routers  web路由
│   │   ├── middleware   中间件
│   │   ├── routers.go  路由初始化
│   │   └── root_router.go 未分组的根路由
│   ├── dal  数据库model query定义
│   │   ├── cmd  使用gen生成model跟query、使用init初始化数据
│   │   ├── model  生成的model
│   │   ├── query  model对应的query
│   │   │   └── gen.go
│   │   ├── repo  db的repo
│   │   │   └── repo.go
│   │   └── query_model.go  需要生成的model列表
│   ├── server  服务相关
│   │   └── http_server.go  http服务启动，关闭
│   └── service 业务处理层
├── logs  日志
├── test  测试用例
├── pkg   公用工具包
│   ├── xauth   认证相关
│   │   └── jwt
│   │       └── jwt.go
│   ├── xcache   缓存相关
│   │   └── redis     redis
│   │       └── redis.go
│   │   └── cache.go    缓存定义
│   │   └── redis_cache.go    基于redis的缓存使用
│   ├── xcolor   带颜色字符串
│   ├── xcryption   加解密，编码等操作
│   ├── xdatabase   数据库相关
│   │   └── mysql     mysql数据库
│   │       └── mysql.go     
│   ├── xerror response自定义错误码  
│   ├── xlimiter 限流相关  
│   ├── xlogger 日志相关  
│   ├── xmq 消息队列(pulsar等)  
│   ├── xrequests http请求相关
│   ├── xresponse 请求统一格式处理
│   ├── xstr_tool 字符串相关操作
│   ├── xlock 分布式锁实现
│   └── common.go  常用工具
├── Makefile
├── Dockerfile
├── go.mod
├── go.sum
└── main.go  项目启动入口
```

------------
### 🚀 快速开始
#### 1. 修改配置
修改配置文件
```shell
vim config$.{env}.yaml
```
根据需要注册mysql、redis等
```
# vim main.go
// 初始化mysql
mysql.InitMysql()
defer mysql.CloseAllMysql(mysql.DB, mysql.DbMap)
// 初始化redis
redis.InitRedis()
defer redis.CloseRedis(redis.RDB)
```

------------
#### 2. 运行项目
![](/assets/images/run.png)
##### 2.1 💻 本地运行
安装运行需要的依赖
```shell
go mod download
go mod tidy
```
初始化项目(初始化数据等)
```shell
make init
```
启动项目
```shell
go run main.go
```

------------
##### 2.2 🐳 Docker 运行
生成项目服务docker镜像
```shell
docker build -t snowgo:v1.0.0 .
```
启动项目
```shell
docker run --name snowgo-service --restart always -d -p 8000:8000 -e ENV=dev -v ./config:/snowgo-service/config -v ./logs:/snowgo-service/logs snow:v1.0.0
```

------------
##### 2.3 🛠 Docker Compose 部署
生成项目服务docker镜像
```shell
docker build -t snowgo:v1.0.0 .
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
docker-compose up -d
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
   - [项目接口 文档](https://apifox.com/apidoc/shared-becb3022-d340-491c-bdd7-1f4d4b84620f)
