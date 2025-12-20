package main

import (
	"context"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"snowgo/config"
	"snowgo/internal/constant"
	"snowgo/pkg/xmq"
	"snowgo/pkg/xmq/rabbitmq"
	"strings"
	"time"
)

func DeclareTopology(ctx context.Context, conn *amqp.Connection) error {
	// 注册mq相关
	reg := rabbitmq.NewRegistry(conn).
		Add(rabbitmq.MQDeclare{
			Name: constant.NormalExchange,
			Type: xmq.DirectExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: constant.ExampleNormalQueue, RoutingKeys: []string{constant.ExampleNormalRoutingKey}},
			},
		}).
		Add(rabbitmq.MQDeclare{
			Name: constant.DelayedExchange,
			Type: xmq.DelayedExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: constant.ExampleDelayedQueue, RoutingKeys: []string{constant.ExampleDelayedRoutingKey}},
			},
		})
	// 构建 log 信息
	var logBuilder strings.Builder
	logBuilder.WriteString("==== MQ Topology Registration ====\n")
	for idx, mq := range reg.Declares {
		logBuilder.WriteString(fmt.Sprintf("[Exchange %d] Name: %s | Type: %s\n", idx+1, mq.Name, mq.Type))
		for qIdx, q := range mq.Queues {
			logBuilder.WriteString(fmt.Sprintf("    [Queue %d] Name: %s | RoutingKeys: %v\n", qIdx+1, q.Name, q.RoutingKeys))
		}
		logBuilder.WriteString("\n")
	}
	log.Println(logBuilder.String())

	// 简单重试：3 次，间隔逐步增长
	var lastErr error
	backoffList := []time.Duration{200 * time.Millisecond, 500 * time.Millisecond, 1000 * time.Millisecond}
	for i := 0; i < len(backoffList); i++ {
		if err := reg.RegisterAll(ctx); err != nil {
			lastErr = err
			select {
			case <-time.After(backoffList[i]):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			return nil
		}
	}
	return lastErr
}

func main() {
	// 初始化配置文件
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config"
	}
	config.Init(configPath)
	// 获取配置
	cfg := config.Get()

	if cfg.RabbitMQ.URL == "" {
		log.Fatal("rabbitmq url required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("dial rabbitmq failed: %v", err)
	}
	defer conn.Close()

	if err := DeclareTopology(ctx, conn); err != nil {
		log.Fatalf("declare topology failed: %v", err)
	}
	log.Println("declare topology success")
}
