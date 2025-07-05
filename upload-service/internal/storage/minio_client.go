package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
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

	result, err := prepareBucket(client, bucketName)
	if err != nil {
		return result, err
	}

	return &MinioClient{
		client:     client,
		bucketName: bucketName,
		endpoint:   endpoint,
	}, nil
}

func prepareBucket(client *minio.Client, bucketName string) (*MinioClient, error) {
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
	return nil, nil
}

func (m *MinioClient) UploadFile(ctx context.Context, objectName string, file io.Reader, size int64) (string, error) {
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, file, size, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return "", fmt.Errorf("erro ao fazer upload: %w", err)
	}

	url := fmt.Sprintf("http://%s/%s/%s", m.endpoint, m.bucketName, objectName)
	return url, nil
}

// UploadString faz upload de uma string como arquivo no MinIO
func (m *MinioClient) UploadString(ctx context.Context, objectName string, content string) error {
	reader := strings.NewReader(content)

	_, err := m.client.PutObject(ctx, m.bucketName, objectName, reader, int64(len(content)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("erro ao fazer upload de string: %w", err)
	}

	return nil
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
