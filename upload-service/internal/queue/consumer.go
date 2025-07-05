package queue

import (
	"context"
	"encoding/json"
	"log"
	"upload-service/internal/models"
	"upload-service/internal/services/video_processing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer representa o consumer de mensagens do RabbitMQ
type Consumer struct {
	channel           *amqp.Channel
	processor         *video_processing.Processor
	publisher         *Publisher
	processingResults chan *models.VideoProcessingResult
}

// NewConsumer cria um novo consumer
func NewConsumer(channel *amqp.Channel, processor *video_processing.Processor, publisher *Publisher) *Consumer {
	return &Consumer{
		channel:           channel,
		processor:         processor,
		publisher:         publisher,
		processingResults: make(chan *models.VideoProcessingResult, 100),
	}
}

// StartProcessing inicia o processamento dos jobs
func (c *Consumer) StartProcessing(ctx context.Context) error {
	// Consumir da fila de entrada
	msgs, err := c.channel.Consume(
		models.InputProcessingQueue, // queue
		"",                          // consumer
		false,                       // auto-ack
		false,                       // exclusive
		false,                       // no-local
		false,                       // no-wait
		nil,                         // args
	)
	if err != nil {
		return err
	}

	// Iniciar goroutine para processar resultados
	go c.processResults(ctx)

	log.Println("🎬 Consumer iniciado - aguardando jobs de processamento...")

	// Processar mensagens
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			go c.handleMessage(msg)
		}
	}
}

// handleMessage processa uma mensagem individual
func (c *Consumer) handleMessage(msg amqp.Delivery) {
	var job models.VideoProcessingJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		log.Printf("❌ Erro ao deserializar job: %v", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("🎬 Processando job: VideoID=%d, UserID=%d", job.VideoID, job.UserID)

	// Processar o vídeo
	result := c.processor.ProcessVideo(&job)

	// Enviar resultado para o canal
	c.processingResults <- result

	// Acknowledgment da mensagem
	msg.Ack(false)
}

// processResults processa os resultados e os envia para a fila
func (c *Consumer) processResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case result := <-c.processingResults:
			if err := c.publisher.PublishProcessingResult(result); err != nil {
				log.Printf("❌ Erro ao publicar resultado: %v", err)
			}
		}
	}
}

// StartResultConsumer inicia o consumer de resultados
func (c *Consumer) StartResultConsumer(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		models.ProcessingResultQueue, // queue
		"",                           // consumer
		false,                        // auto-ack
		false,                        // exclusive
		false,                        // no-local
		false,                        // no-wait
		nil,                          // args
	)
	if err != nil {
		return err
	}

	log.Println("📋 Consumer de resultados iniciado...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			go c.handleResultMessage(msg)
		}
	}
}

// handleResultMessage processa uma mensagem de resultado
func (c *Consumer) handleResultMessage(msg amqp.Delivery) {
	var result models.VideoProcessingResult
	if err := json.Unmarshal(msg.Body, &result); err != nil {
		log.Printf("❌ Erro ao deserializar resultado: %v", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("📋 Processando resultado: JobID=%s, VideoID=%d, Status=%s",
		result.JobID, result.VideoID, result.Status)

	// Aqui você pode adicionar lógica para atualizar o status na API
	// Por exemplo, chamar uma API para atualizar o status do vídeo

	msg.Ack(false)
}
