package main

import (
	"os"
	"os/signal"
	"snowgo/config"
	"snowgo/routers"
	"snowgo/utils/database/mysql"
	"snowgo/utils/database/redis"
	"snowgo/utils/logger"
	"syscall"
)

func init() {
	// 初始化配置文件
	config.InitConf(
		config.WithMysqlConf(), // 加载mysql配置
		config.WithRedisConf(), // 加载redis配置
		config.WithJwtConf(),   // 加载jwt配置
	)

	// 初始化zap log全局配置
	logger.InitLogger()
}

func main() {

	// 初始化mysql
	mysql.InitMysql()
	defer mysql.CloseMysql(mysql.DB)
	// 初始化redis
	redis.InitRedis()
	defer redis.CloseRedis(redis.RDB)

	// 启动服务
	routers.StartHttpServer()

	// 等待中断信号来优雅地关闭服务器，为关闭服务器操作设置一个超时
	// kill -2 发送 syscall.SIGINT 信号，用户发送INTR字符(Ctrl+C)触发
	// kill -3 发送 syscall.SIGQUIT 信号，用户发送QUIT字符(Ctrl+/)触发
	// kill -15 发送 syscall.SIGTERM 信号，结束程序(可以被捕获、阻塞或忽略)
	// kill -9 发送 syscall.SIGKILL 信号，但是不能被捕获，所以不需要添加它
	quit := make(chan os.Signal, 1)
	// signal.Notify接受信号转发给quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-quit

	// 关闭服务
	_ = routers.StopHttpServer()
}
