package main

import (
	"context"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"snowgo/config"
	"snowgo/internal/worker"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xmq/rabbitmq"
	"syscall"
	"time"
)

func main() {
	// 初始化配置文件
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config"
	}
	config.Init(configPath)

	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "./logs"
	}

	// 获取配置
	cfg := config.Get()
	logger := xlogger.NewLogger(logPath, "rabbitmq-consumer", xlogger.WithFileMaxAgeDays(30))

	//container, err := di.NewContainer(
	//	di.WithMySQL(cfg.Mysql, cfg.OtherDB),
	//	di.WithRedis(cfg.Redis),
	//)
	//if err != nil {
	//	xlogger.Fatalf("new container failed: %v", err)
	//}

	consumerCfg := rabbitmq.NewConsumerConnConfig(
		cfg.RabbitMQ.URL,
		rabbitmq.WithConsumerLogger(logger),
		rabbitmq.WithConsumerReconnectInitialDelay(cfg.RabbitMQ.ReconnectInitialDelayTime),
		rabbitmq.WithConsumerReconnectMaxDelay(cfg.RabbitMQ.ReconnectMaxDelayTime),
	)
	consumer, err := rabbitmq.NewConsumer(context.Background(), consumerCfg)
	if err != nil {
		panic(err)
	}

	// 注册所有队列
	if err := worker.RegisterAll(context.Background(), consumer, &worker.Deps{
		Logger: logger,
	}); err != nil {
		panic(err)
	}

	// 启动
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := consumer.Start(ctx); err != nil {
			logger.Error(ctx, "consumer start fail", zap.Error(err))
		}
	}()

	<-ctx.Done()
	// 优雅退出
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := consumer.Stop(shutdownCtx); err != nil {
		logger.Error(shutdownCtx, "consumer stop fail", zap.Error(err))
	}
	// 注入关闭
	//if err := container.Close(); err != nil {
	//	xlogger.Errorf("container close error: %v", err)
	//}
	logger.Info(shutdownCtx, "worker exit")
}
