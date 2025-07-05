package queue

import (
	"log"
	"upload-service/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQClient representa o cliente RabbitMQ
type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  *config.Config
}

// NewRabbitMQClient cria uma nova conexão com RabbitMQ
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

	// Declarar as filas
	if err := client.declareQueues(); err != nil {
		conn.Close()
		return nil, err
	}

	log.Println("✅ Conectado ao RabbitMQ")
	return client, nil
}

// declareQueues declara as filas necessárias
func (r *RabbitMQClient) declareQueues() error {
	// Fila de entrada para processamento
	_, err := r.channel.QueueDeclare(
		"input_processing_queue", // name
		true,                     // durable
		false,                    // delete when unused
		false,                    // exclusive
		false,                    // no-wait
		nil,                      // arguments
	)
	if err != nil {
		return err
	}

	return nil
}

// Close fecha a conexão com RabbitMQ
func (r *RabbitMQClient) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// GetChannel retorna o canal do RabbitMQ
func (r *RabbitMQClient) GetChannel() *amqp.Channel {
	return r.channel
}
