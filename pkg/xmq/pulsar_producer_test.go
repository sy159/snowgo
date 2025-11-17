package xmq

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"sync"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
)

// 测试默认队列
func TestDefaultQueue(t *testing.T) {
	client, err := NewPulsarClient("pulsar://localhost:6650")
	if err != nil {
		t.Fatalf("Failed to client pulsar: %v", err)
	}
	p, err := client.NewProducer("persistent://snow/test/test-topic")
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()
	defer client.Close()
	message := map[string]string{"name": "test", "age": "18"}
	messageBytes, _ := json.Marshal(message)

	t.Run("SyncSend", func(t *testing.T) {
		err := p.SendMessage(context.Background(), messageBytes, nil)
		assert.NoError(t, err, "Failed to send message synchronously")
	})

	t.Run("AsyncSend", func(t *testing.T) {
		callback := func(id any, msg interface{}, err error) {
			assert.NoError(t, err, "Failed to send delayed message asynchronously")
			// 可选类型断言
			if mid, ok := id.(pulsar.MessageID); ok {
				fmt.Println("MessageID:", mid.String())
			}
		}
		p.SendAsyncMessage(context.Background(), messageBytes, nil, callback)
		// Add a short sleep to wait for the async callback to complete
		time.Sleep(1 * time.Second)
	})
}

// 测试延时队列
func TestDelayedQueue(t *testing.T) {
	client, err := NewPulsarClient("pulsar://localhost:6650")
	if err != nil {
		t.Fatalf("Failed to client pulsar: %v", err)
	}
	p, err := client.NewProducer("persistent://snow/test/delay-topic", WithDeliverAfter(5*time.Second))
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()
	defer client.Close()
	message := map[string]string{"name": "test", "age": "18"}
	messageBytes, _ := json.Marshal(message)

	t.Run("SyncSend", func(t *testing.T) {
		err := p.SendMessage(context.Background(), messageBytes, nil)
		assert.NoError(t, err, "Failed to send delayed message synchronously")
	})

	t.Run("AsyncSend", func(t *testing.T) {
		callback := func(id any, msg interface{}, err error) {
			assert.NoError(t, err, "Failed to send delayed message asynchronously")
			// 可选类型断言
			if mid, ok := id.(pulsar.MessageID); ok {
				fmt.Println("MessageID:", mid.String())
			}
		}
		p.SendAsyncMessage(context.Background(), messageBytes, nil, callback)
		// Add a short sleep to wait for the async callback to complete
		time.Sleep(1 * time.Second)
	})
}

// 测试批量队列
func TestBatchingQueue(t *testing.T) {
	client, err := NewPulsarClient("pulsar://localhost:6650")
	if err != nil {
		t.Fatalf("Failed to client pulsar: %v", err)
	}
	p, err := client.NewProducer("persistent://snow/test/batch-topic", WithMessageBatching(
		false,
		1*time.Second,
		50,
		1024*1024,
	))
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()
	message := map[string]string{"name": "test", "age": "18"}
	messageBytes, _ := json.Marshal(message)
	const testCount = 100

	t.Run("SyncSend", func(t *testing.T) {
		// 同步测试
		for i := 0; i < testCount; i++ {
			err := p.SendMessage(context.Background(), messageBytes, nil)
			assert.NoError(t, err, "Sync send failed")
		}
	})

	t.Run("AsyncSend", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(testCount)
		callback := func(id any, msg interface{}, err error) {
			defer wg.Done()
			var msgIDStr string
			if mid, ok := id.(pulsar.MessageID); ok {
				msgIDStr = mid.String()
			} else {
				msgIDStr = fmt.Sprintf("%v", id) // fallback
			}

			// 类型断言 msg
			var payloadStr string
			var properties map[string]string
			if pm, ok := msg.(*pulsar.ProducerMessage); ok {
				payloadStr = string(pm.Payload)
				properties = pm.Properties
			} else {
				payloadStr = fmt.Sprintf("%v", msg)
				properties = nil
			}
			producerLogger.Log(
				"pulsar producer",
				zap.String("topic", p.producer.Topic()),
				zap.String("message_id", msgIDStr),
				zap.String("message", payloadStr),
				zap.Any("properties", properties),
				zap.String("status", "success"),
				zap.String("error_msg", ""),
			)
			assert.NoError(t, err, "Failed to send batched message asynchronously")
		}
		for i := 0; i < testCount; i++ {
			p.SendAsyncMessage(context.Background(), messageBytes, nil, callback)
		}
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("Async sends timeout")
		}
	})
}
