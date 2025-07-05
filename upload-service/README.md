# Serviço de Upload com Processamento de Vídeo

Este serviço gerencia o upload de vídeos e os envia para processamento através de filas RabbitMQ. Os vídeos processados são salvos diretamente no MinIO.

## 🏗️ Arquitetura

```
upload-service/
├── cmd/
│   └── main.go                    # Ponto de entrada da aplicação
├── internal/
│   ├── config/
│   │   └── config.go              # Configurações centralizadas
│   ├── middleware/
│   │   └── auth.go                # Middleware de autenticação JWT
│   ├── models/
│   │   └── video_processing.go    # Modelos de dados
│   ├── queue/
│   │   ├── rabbitmq.go            # Cliente RabbitMQ
│   │   ├── publisher.go           # Publisher de mensagens
│   │   └── consumer.go            # Consumer de mensagens
│   ├── services/
│   │   ├── upload/
│   │   │   └── upload.go          # Serviço de upload
│   │   └── video_processing/
│   │       └── processor.go       # Processador de vídeos
│   └── storage/
│       └── minio_client.go        # Cliente MinIO
├── go.mod
└── go.sum
```

## 🔄 Fluxo de Processamento

1. **Upload**: Usuário faz upload do vídeo
2. **Armazenamento**: Vídeo é salvo no MinIO em `{user_id}/input/{filename}`
3. **Registro**: Entidade é criada na API principal
4. **Envio para Fila**: Job é enviado para `input_processing_queue`
5. **Processamento**: Consumer processa o vídeo
6. **Salvamento**: Vídeo processado é salvo em `{user_id}/outputs/{processed_filename}`

## 📋 Filas RabbitMQ

- **`input_processing_queue`**: Recebe jobs de processamento

## 📁 Estrutura de Pastas no MinIO

```
videos/
├── {user_id}/
│   ├── input/
│   │   ├── video1.mp4
│   │   └── video2.avi
│   └── outputs/
│       ├── video1_processed.mp4
│       └── video2_processed.mp4
```

## 🌐 Acessos

### RabbitMQ Management
- **URL**: http://localhost:15672
- **Usuário**: guest
- **Senha**: guest

### MinIO Console
- **URL**: http://localhost:9001
- **Usuário**: minioadmin
- **Senha**: minioadmin

## ⚙️ Configuração

### Variáveis de Ambiente

```bash
# MinIO
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=videos

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

# API
API_BASE_URL=http://localhost:8080

# Server
SERVER_PORT=8081
```

## 🚀 Execução

### Com Docker Compose (Recomendado)
```bash
# Subir todos os serviços
docker-compose up -d

# Ver logs do upload-service
docker-compose logs -f upload-service
```

### Local
```bash
# Instalar dependências
go mod tidy

# Executar
go run cmd/main.go
```

## 📊 Status do Processamento

- `pending`: Aguardando processamento
- `processing`: Em processamento
- `completed`: Processamento concluído
- `failed`: Processamento falhou

## 🔧 Próximos Passos

1. **Implementar processamento real**: Substituir a simulação no `processor.go`
2. **Adicionar retry**: Implementar retry para jobs que falharam
3. **Métricas**: Adicionar métricas de processamento
4. **Logs estruturados**: Implementar logs estruturados
5. **Testes**: Adicionar testes unitários e de integração 