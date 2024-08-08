package mq

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
)

func TestPulsarConsumer_SameSubscriptionName(t *testing.T) {
	url := "pulsar://localhost:6650"
	topic := "persistent://snow/test/test-topic"
	subscriptionName := "test-subscription"

	// 启动两个消费者，使用相同的SubscriptionName
	consumer1, err := NewPulsarConsumer(url, topic, testHandler, WithSubscriptionName(subscriptionName), WithRunNum(2))
	assert.NoError(t, err)
	defer consumer1.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumer1.Start(ctx)

	// 等待一段时间以确保消费者开始消费
	time.Sleep(20 * time.Second)
}

func TestPulsarConsumer_DifferentSubscriptionNames_SingleConsumer(t *testing.T) {
	url := "pulsar://localhost:6650"
	topic := "persistent://snow/test/test-topic"

	// 启动两个消费者，使用不同的SubscriptionName，每个消费一个
	consumer1, err := NewPulsarConsumer(url, topic, testHandler, WithSubscriptionName("sub1"))
	assert.NoError(t, err)
	defer consumer1.Close()

	consumer2, err := NewPulsarConsumer(url, topic, testHandler, WithSubscriptionName("sub2"))
	assert.NoError(t, err)
	defer consumer2.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumer1.Start(ctx)
	go consumer2.Start(ctx)

	// 等待一段时间以确保消费者开始消费
	time.Sleep(10 * time.Second)
}

func TestPulsarConsumer_DifferentSubscriptionNames_MultipleConsumers(t *testing.T) {
	url := "pulsar://localhost:6650"
	topic := "persistent://snow/test/test-topic"

	// 启动两个消费者组，分别使用不同的SubscriptionName，每组启动两个消费者
	consumerGroup1, err := NewPulsarConsumer(url, topic, testHandler, WithSubscriptionName("sub1"), WithRunNum(2))
	assert.NoError(t, err)
	defer consumerGroup1.Close()

	consumerGroup2, err := NewPulsarConsumer(url, topic, testHandler, WithSubscriptionName("sub2"), WithRunNum(2))
	assert.NoError(t, err)
	defer consumerGroup2.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumerGroup1.Start(ctx)
	go consumerGroup2.Start(ctx)

	// 等待一段时间以确保消费者开始消费
	time.Sleep(10 * time.Second)
}

// testHandler 处理测试消息
func testHandler(ctx context.Context, msg pulsar.Message) {
	fmt.Printf("Received a message: %s; properties: %+v; msg id: %s\n", string(msg.Payload()), msg.Properties(), msg.ID())
	time.Sleep(1 * time.Second)
}
