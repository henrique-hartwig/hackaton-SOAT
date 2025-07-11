package queue

import (
	"log"
	"src/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  *config.Config
}

func NewRabbitMQClient(cfg *config.Config) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	client := &RabbitMQClient{
		conn:    conn,
		channel: ch,
		config:  cfg,
	}

	if err := client.declareQueues(); err != nil {
		conn.Close()
		return nil, err
	}

	log.Println("✅ Conectado ao RabbitMQ")
	return client, nil
}

func (r *RabbitMQClient) declareQueues() error {
	_, err := r.channel.QueueDeclare(
		"input_processing_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQClient) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func (r *RabbitMQClient) GetChannel() *amqp.Channel {
	return r.channel
}
