package rabbitmq

import (
	"context"
	"fmt"
	"snowgo/pkg/xmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueDeclare Queue 简化声明结构
type QueueDeclare struct {
	Name        string
	RoutingKeys []string
	Args        amqp.Table // 留扩展口
}

// MQDeclare Exchange 简化声明结构
type MQDeclare struct {
	Name   string
	Type   xmq.ExchangeType
	Queues []QueueDeclare
}

// Registry 简化注册器
type Registry struct {
	Conn     *amqp.Connection
	Declares []MQDeclare
}

func NewRegistry(conn *amqp.Connection) *Registry {
	return &Registry{
		Conn:     conn,
		Declares: make([]MQDeclare, 0),
	}
}

func (r *Registry) Add(declare MQDeclare) *Registry {
	r.Declares = append(r.Declares, declare)
	return r
}

// RegisterAll 注册所有 Exchange + Queue + Binding
func (r *Registry) RegisterAll(ctx context.Context) error {
	ch, err := r.Conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	for _, ex := range r.Declares {
		// 延时交换机需要特殊 arg
		args := amqp.Table{}
		if ex.Type == xmq.DelayedExchange {
			args["x-delayed-type"] = "direct"
		}

		// 声明 Exchange
		err = ch.ExchangeDeclare(
			ex.Name,
			string(ex.Type),
			true,
			false, // autoDelete
			false, // internal
			false, // noWait
			args,
		)
		if err != nil {
			return fmt.Errorf("declare exchange %s: %w", ex.Name, err)
		}

		// Queue + Binding
		for _, q := range ex.Queues {

			_, err := ch.QueueDeclare(
				q.Name,
				true,
				false,
				false,
				false,
				q.Args,
			)
			if err != nil {
				return fmt.Errorf("declare queue %s: %w", q.Name, err)
			}

			for _, rk := range q.RoutingKeys {
				err = ch.QueueBind(
					q.Name,
					rk,
					ex.Name,
					false,
					nil,
				)
				if err != nil {
					return fmt.Errorf("bind queue=%s routingKey=%s exchange=%s: %w", q.Name, rk, ex.Name, err)
				}
			}

		}
	}

	return nil
}
