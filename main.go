package main

import (
	"os"
	"os/signal"
	"snowgo/config"
	"snowgo/internal/server"
	"snowgo/pkg/xdatabase/mysql"
	"snowgo/pkg/xdatabase/redis"
	"snowgo/pkg/xlogger"
	"syscall"
)

func init() {
	// 初始化配置文件
	config.Init("./config")

	// 初始化zap log全局配置
	xlogger.InitLogger()
}

func main() {

	// 初始化mysql
	mysql.InitMysql()
	defer mysql.CloseAllMysql(mysql.DB, mysql.DbMap)
	// 初始化redis
	redis.InitRedis()
	defer redis.CloseRedis(redis.RDB)

	// 启动服务
	server.StartHttpServer()

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
	_ = server.StopHttpServer()
}
