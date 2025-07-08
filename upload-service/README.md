# Upload Service

Serviço de upload e processamento de vídeos com arquitetura assíncrona usando Go, RabbitMQ, MinIO, PostgreSQL e Redis.

## 🏗️ Arquitetura

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Frontend  │    │   API       │    │ Upload      │
│   (Nginx)   │◄──►│   (Go)      │◄──►│ Service     │
└─────────────┘    └─────────────┘    └─────────────┘
                           │                   │
                           ▼                   ▼
                   ┌─────────────┐    ┌─────────────┐
                   │ PostgreSQL  │    │   RabbitMQ  │
                   │   (DB)      │    │   (Queue)   │
                   └─────────────┘    └─────────────┘
                                              │
                                              ▼
                                     ┌─────────────┐
                                     │   MinIO     │
                                     │ (Storage)   │
                                     └─────────────┘
                                              │
                                              ▼
                                     ┌─────────────┐
                                     │   Redis     │
                                     │   (Cache)   │
                                     └─────────────┘
```

## 🚀 Funcionalidades

### ✅ Implementadas

1. **Upload de Vídeos**
   - Upload via API REST
   - Validação de tipos de arquivo
   - Armazenamento no MinIO
   - Integração com API de vídeos

2. **Processamento Assíncrono**
   - Jobs enviados para RabbitMQ
   - Consumer processa vídeos em background
   - Resultados salvos no MinIO (pasta outputs)

3. **Sistema de Retry**
   - Até 3 tentativas de processamento
   - Backoff exponencial entre tentativas
   - Atualização automática de status via API

4. **Cache Redis**
   - Cache de metadados de vídeos (TTL: 1h)
   - Cache de sessões de usuário (TTL: 24h)
   - Cache de status de processamento (TTL: 10min)
   - Cache de dados de usuário (TTL: 30min)

5. **Atualização de Status**
   - Status atualizado via API REST
   - Mapeamento automático de status
   - Logs detalhados do processamento

### 🔄 Fluxo de Processamento

1. **Upload**: Usuário faz upload → arquivo salvo no MinIO → job enviado para RabbitMQ
2. **Processamento**: Consumer pega job → processa vídeo → salva resultado no MinIO
3. **Status**: Status atualizado na API → cache Redis atualizado
4. **Retry**: Se falhar, tenta novamente com backoff exponencial

## 🛠️ Tecnologias

- **Go 1.23** - Linguagem principal
- **Gin** - Framework web
- **RabbitMQ** - Message broker
- **MinIO** - Object storage
- **Redis** - Cache
- **PostgreSQL** - Banco de dados (via API)
- **Docker** - Containerização

## 📦 Estrutura do Projeto

```
upload-service/
├── cmd/
│   └── main.go              # Ponto de entrada
├── internal/
│   ├── cache/
│   │   └── redis.go         # Cliente Redis
│   ├── config/
│   │   └── config.go        # Configurações
│   ├── middleware/
│   │   └── auth.go          # Middleware de autenticação
│   ├── models/
│   │   └── video_processing.go # Modelos de dados
│   ├── queue/
│   │   ├── consumer.go      # Consumer RabbitMQ
│   │   ├── publisher.go     # Publisher RabbitMQ
│   │   └── rabbitmq.go      # Cliente RabbitMQ
│   ├── services/
│   │   ├── upload/
│   │   │   └── upload.go    # Lógica de upload
│   │   └── video_processing/
│   │       └── processor.go # Processamento de vídeo
│   └── storage/
│       └── minio_client.go  # Cliente MinIO
├── test-integration.go      # Testes de integração
├── go.mod                   # Dependências
└── README.md               # Documentação
```

## 🔧 Configuração

### Variáveis de Ambiente

```bash
# MinIO
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=videos

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

# Redis
REDIS_URL=redis://redis:6379

# API
API_BASE_URL=http://api:8080

# Server
SERVER_PORT=8081
```

### Docker Compose

```bash
# Subir todos os serviços
docker-compose up -d

# Ver logs do upload-service
docker-compose logs -f upload-service

# Parar todos os serviços
docker-compose down
```

## 🧪 Testes

### Teste de Integração

```bash
# Executar teste de integração
go run test-integration.go
```

Este teste verifica:
- ✅ Conexão com MinIO
- ✅ Operações de cache Redis
- ✅ Conexão com RabbitMQ
- ✅ Publicação de jobs
- ✅ Processamento de vídeos
- ✅ Cache de sessões e status

## 📊 Status de Processamento

### Estados do Vídeo

- **pending**: Aguardando processamento
- **processing**: Em processamento
- **processed**: Processado com sucesso
- **failed**: Falha no processamento

### Mapeamento de Status

| Upload Service | API Status |
|----------------|------------|
| pending        | pending    |
| processing     | pending    |
| completed      | processed  |
| failed         | failed     |

## 🔄 Retry Logic

### Configuração

- **Máximo de tentativas**: 3
- **Backoff**: Exponencial (1s, 4s, 9s)
- **Timeout**: 10 segundos por requisição

### Fluxo de Retry

1. Tentativa 1: Processa vídeo
2. Se falhar: Aguarda 1 segundo
3. Tentativa 2: Processa vídeo novamente
4. Se falhar: Aguarda 4 segundos
5. Tentativa 3: Última tentativa
6. Se falhar: Status atualizado para "failed"

## 💾 Cache Redis

### Estruturas de Cache

```go
// Cache de vídeo
type VideoCache struct {
    ID          uint      `json:"id"`
    Title       string    `json:"title"`
    Status      string    `json:"status"`
    UserID      uint      `json:"user_id"`
    URL         string    `json:"url"`
    Duration    int       `json:"duration,omitempty"`
    Thumbnail   string    `json:"thumbnail,omitempty"`
    ProcessedAt time.Time `json:"processed_at,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

// Cache de sessão
type UserSession struct {
    UserID    uint      `json:"user_id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    Roles     []string  `json:"roles"`
    LastLogin time.Time `json:"last_login"`
}

// Cache de status de processamento
type ProcessingStatus struct {
    VideoID       uint      `json:"video_id"`
    Status        string    `json:"status"`
    Progress      int       `json:"progress"`
    Message       string    `json:"message"`
    EstimatedTime int       `json:"estimated_time"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

### TTLs Configurados

- **Vídeos**: 1 hora
- **Sessões**: 24 horas
- **Status de processamento**: 10 minutos
- **Dados de usuário**: 30 minutos

## 🚀 Próximos Passos

### Implementações Futuras

1. **Sistema de Métricas**
   - Prometheus + Grafana
   - Métricas de processamento
   - Dashboards de monitoramento

2. **Melhorias no Cache**
   - Cache de listas de vídeos
   - Invalidação inteligente
   - Compressão de dados

3. **Processamento Avançado**
   - Transcodificação de vídeo
   - Geração de thumbnails
   - Análise de metadados

4. **Monitoramento**
   - Health checks avançados
   - Alertas automáticos
   - Logs estruturados

## 📝 Logs

### Exemplos de Logs

```
🎬 Consumer iniciado - aguardando jobs de processamento...
🎬 Processando job: VideoID=1, UserID=1
🔄 Tentativa 1/3 para VideoID=1
✅ Status atualizado para VideoID=1: pending
✅ Vídeo processado salvo: 1/outputs/test_video_processed.mp4
✅ Status atualizado para VideoID=1: processed
```

## 🤝 Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature
3. Commit suas mudanças
4. Push para a branch
5. Abra um Pull Request

## 📄 Licença

Este projeto está sob a licença MIT. 