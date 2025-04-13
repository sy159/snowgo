package xlogger

import (
	"go.uber.org/zap"
	"testing"
)

func TestLog(t *testing.T) {
	t.Run("producer log", func(t *testing.T) {
		logger := NewLogger("pulsar-producer")
		for i := 0; i < 10; i++ {
			logger.Log(
				" producer test",
				zap.String("topic", "test-topic"),
				zap.Int("msgId", i),
			)
		}
	})

	t.Run("consumer log", func(t *testing.T) {
		logger := NewLogger("pulsar-consumer")
		for i := 0; i < 10; i++ {
			logger.Log(
				" consumer test",
				zap.String("topic", "test-topic"),
				zap.Int("msgId", i),
			)
		}
	})
}

func TestConsoleLog(t *testing.T) {
	t.Run("producer log", func(t *testing.T) {
		logger := NewLogger("pulsar-producer", WithConsoleOutput(true), WithFileMaxAgeDays(3))
		for i := 0; i < 10; i++ {
			logger.Log(
				" producer test",
				zap.String("topic", "test-topic"),
				zap.Int("msgId", i),
			)
		}
	})

	t.Run("consumer log", func(t *testing.T) {
		logger := NewLogger("pulsar-consumer", WithConsoleOutput(true), WithFileMaxAgeDays(3))
		for i := 0; i < 10; i++ {
			logger.Log(
				" consumer test",
				zap.String("topic", "test-topic"),
				zap.Int("msgId", i),
			)
		}
	})
}
