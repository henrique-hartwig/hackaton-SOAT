package queue

import (
	"encoding/json"
	"log"
	"upload-service/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher representa o publisher de mensagens para RabbitMQ
type Publisher struct {
	channel *amqp.Channel
}

// NewPublisher cria um novo publisher
func NewPublisher(channel *amqp.Channel) *Publisher {
	return &Publisher{
		channel: channel,
	}
}

// PublishVideoProcessingJob publica um job de processamento de vídeo
func (p *Publisher) PublishVideoProcessingJob(job *models.VideoProcessingJob) error {
	// Serializar o job para JSON
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}

	// Publicar na fila de processamento
	err = p.channel.Publish(
		"",                          // exchange
		models.InputProcessingQueue, // routing key
		false,                       // mandatory
		false,                       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jobBytes,
		},
	)

	if err != nil {
		return err
	}

	log.Printf("📤 Job enviado para processamento: VideoID=%d, UserID=%d", job.VideoID, job.UserID)
	return nil
}

// PublishProcessingResult publica o resultado do processamento
func (p *Publisher) PublishProcessingResult(result *models.VideoProcessingResult) error {
	// Serializar o resultado para JSON
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	// Publicar na fila de resultados
	err = p.channel.Publish(
		"",                           // exchange
		models.ProcessingResultQueue, // routing key
		false,                        // mandatory
		false,                        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        resultBytes,
		},
	)

	if err != nil {
		return err
	}

	log.Printf("📤 Resultado enviado: JobID=%s, VideoID=%d, Status=%s", result.JobID, result.VideoID, result.Status)
	return nil
}
