# snowgo
基于gin开发的脚手架工具，封装了常用功能，便于快速开发接口，开箱即用。

### 集成组建:
1. gin轻量级Web框架
2. zap日志管理
3. viper配置文件解析
4. response统一结构返回，以及error code自定义
5. gorm数据库组件，以及使用gen生成model以及query(支持读写分离以及多数据库配置)
6. go-redis缓存组件
7. jwt鉴权
8. rate限流
9. 访问日志、跨域、全局异常处理等中间件

### 目录结构
```
snowgo
├── config  配置文件
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
│   ├── dal  数据库model query定义
│   │   ├── cmd  使用gen生成model跟query
│   │   │   └── gen.go
│   │   ├── model  生成的model
│   │   ├── query  model对应的query
│   │   │   └── gen.go
│   │   ├── repo  db的repo
│   │   │   └── repo.go
│   │   └── query_model.go  需要生成的model列表
│   └── service 业务处理层
├── logs  日志
├── routers  web路由
│   ├── middleware   中间件
│   ├── routers.go  路由初始化
│   ├── http_server.go  服务启动，关闭
│   └── rouut_router.go 未分组的根路由
├── test  测试用例
├── utils   公用工具包
│   ├── auth   认证相关
│   │   └── jwt
│   │       └── jwt.go
│   ├── cache   缓存相关
│   │   └── redis     redis
│   │       └── redis.go
│   ├── color   带颜色字符串
│   ├── cryption   加解密，编码等操作
│   ├── database   数据库相关
│   │   └── msyql     mysql数据库
│   │       └── mysql.go     
│   ├── error response自定义错误码  
│   ├── limiter 限流相关  
│   ├── logger 日志相关  
│   ├── requests http请求相关
│   ├── response 请求统一格式处理
│   ├── str_tool 字符串相关操作
│   └── common.go  常用工具
├── Makefile
├── Dockerfile
├── go.mod
├── go.sum
└── main.go  项目启动入口

```

### 安装部署
安装运行需要的依赖
```
go mod download
go mod tidy
```
修改配置文件
```
vim config$.{env}.yaml
```
根据需要注册mysql、redis等
```
// 初始化mysql
mysql.InitMysql()
defer mysql.CloseMysql(mysql.DB)
// 初始化redis
redis.InitRedis()
defer redis.CloseRedis(redis.RDB)
```
启动项目
```
go run main.go
```

### 注意事项
1. 根据数据库表生成model
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
2. 数据库orm语句
    参考: [gen](https://gorm.io/zh_CN/gen/dao.html)、[gorm](https://gorm.io/zh_CN/docs/)
