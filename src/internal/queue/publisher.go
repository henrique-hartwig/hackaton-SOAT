package queue

import (
	"encoding/json"
	"log"
	"src/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	channel *amqp.Channel
}

func NewPublisher(channel *amqp.Channel) *Publisher {
	return &Publisher{
		channel: channel,
	}
}

func (p *Publisher) PublishVideoProcessingJob(job *models.VideoProcessingJob) error {
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}

	err = p.channel.Publish(
		"",
		models.InputProcessingQueue,
		false,
		false,
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
