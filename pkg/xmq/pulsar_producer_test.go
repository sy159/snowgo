package xmq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
)

// 测试默认队列
func TestDefaultQueue(t *testing.T) {
	p, err := NewPulsarProducer("pulsar://localhost:6650", "persistent://snow/test/test-topic")
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()
	message := map[string]string{"name": "test", "age": "18"}
	messageBytes, _ := json.Marshal(message)

	t.Run("SyncSend", func(t *testing.T) {
		err := p.SendMessage(context.Background(), messageBytes, nil)
		assert.NoError(t, err, "Failed to send message synchronously")
	})

	t.Run("AsyncSend", func(t *testing.T) {
		callback := func(id pulsar.MessageID, msg *pulsar.ProducerMessage, err error) {
			assert.NoError(t, err, "Failed to send message asynchronously")
		}
		p.SendAsyncMessage(context.Background(), messageBytes, nil, callback)
		// Add a short sleep to wait for the async callback to complete
		time.Sleep(1 * time.Second)
	})
}

// 测试延时队列
func TestDelayedQueue(t *testing.T) {
	p, err := NewPulsarProducer(
		"pulsar://localhost:6650",
		"persistent://snow/test/delay-topic",
		WithDeliverAfter(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()
	message := map[string]string{"name": "test", "age": "18"}
	messageBytes, _ := json.Marshal(message)

	t.Run("SyncSend", func(t *testing.T) {
		err := p.SendMessage(context.Background(), messageBytes, nil)
		assert.NoError(t, err, "Failed to send delayed message synchronously")
	})

	t.Run("AsyncSend", func(t *testing.T) {
		callback := func(id pulsar.MessageID, msg *pulsar.ProducerMessage, err error) {
			assert.NoError(t, err, "Failed to send delayed message asynchronously")
		}
		p.SendAsyncMessage(context.Background(), messageBytes, nil, callback)
		// Add a short sleep to wait for the async callback to complete
		time.Sleep(1 * time.Second)
	})
}

// 测试批量队列
func TestBatchingQueue(t *testing.T) {
	p, err := NewPulsarProducer(
		"pulsar://localhost:6650",
		"persistent://snow/test/batch-topic",
		WithMessageBatching(
			false,
			1*time.Second,
			10,
			1024*1024,
		),
	)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()
	message := map[string]string{"name": "test", "age": "18"}
	messageBytes, _ := json.Marshal(message)

	t.Run("SyncSend", func(t *testing.T) {
		err := p.SendMessage(context.Background(), messageBytes, nil)
		assert.NoError(t, err, "Failed to send batched message synchronously")
	})

	t.Run("AsyncSend", func(t *testing.T) {
		callback := func(id pulsar.MessageID, msg *pulsar.ProducerMessage, err error) {
			assert.NoError(t, err, "Failed to send batched message asynchronously")
		}
		p.SendAsyncMessage(context.Background(), messageBytes, nil, callback)
		// Add a short sleep to wait for the async callback to complete
		time.Sleep(1 * time.Second)
	})
}
