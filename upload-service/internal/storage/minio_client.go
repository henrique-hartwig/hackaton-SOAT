package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	client     *minio.Client
	bucketName string
	endpoint   string
}

func NewMinioClient(endpoint, accessKey, secretKey, bucketName string) (*MinioClient, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar no Minio: %w", err)
	}

	// Verificar se bucket existe, se não, criar
	exists, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("erro ao criar bucket: %w", err)
		}
	}

	return &MinioClient{
		client:     client,
		bucketName: bucketName,
		endpoint:   endpoint,
	}, nil
}

func (m *MinioClient) UploadFile(ctx context.Context, objectName string, file io.Reader, size int64) (string, error) {
	// Upload do arquivo
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, file, size, minio.PutObjectOptions{
		ContentType: "video/mp4", // Pode ser ajustado baseado no tipo de arquivo
	})
	if err != nil {
		return "", fmt.Errorf("erro ao fazer upload: %w", err)
	}

	// Gerar URL de acesso
	url := fmt.Sprintf("http://%s/%s/%s", m.endpoint, m.bucketName, objectName)
	return url, nil
}

func (m *MinioClient) DeleteFile(ctx context.Context, objectName string) error {
	return m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
}

func (m *MinioClient) GetFileURL(objectName string, expires time.Duration) (string, error) {
	url, err := m.client.PresignedGetObject(context.Background(), m.bucketName, objectName, expires, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar URL: %w", err)
	}
	return url.String(), nil
}
