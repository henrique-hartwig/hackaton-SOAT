package video_processing

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"src/internal/models"
	"src/internal/storage"
	"strings"
	"time"
)

// ProcessingResult representa o resultado do processamento
type ProcessingResult struct {
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	ProcessedAt time.Time `json:"processed_at"`
	ZipPath     string    `json:"zip_path,omitempty"`
	FrameCount  int       `json:"frame_count,omitempty"`
	Images      []string  `json:"images,omitempty"`
}

// Processor representa o processador de vídeos
type Processor struct {
	// Aqui você pode adicionar dependências como:
	// - Cliente para APIs externas
	// - Configurações de processamento
	// - etc.
	minioClient *storage.MinioClient
}

// NewProcessor cria um novo processador
func NewProcessor() *Processor {
	return &Processor{}
}

// NewProcessorWithMinIO cria um novo processador com cliente MinIO
func NewProcessorWithMinIO(minioClient *storage.MinioClient) *Processor {
	return &Processor{
		minioClient: minioClient,
	}
}

// ProcessVideo processa um vídeo extraindo frames e criando ZIP
func (p *Processor) ProcessVideo(job *models.VideoProcessingJob) *ProcessingResult {
	log.Printf("🎬 Iniciando processamento do vídeo: %s", job.FileName)

	userTempDir := filepath.Join("videos", fmt.Sprintf("%d", job.UserID), "temp")
	os.MkdirAll(userTempDir, 0755)
	os.MkdirAll("outputs", 0755)

	// Gerar timestamp único para o processamento
	timestamp := time.Now().Format("20060102_150405")

	// Baixar vídeo do MinIO para a pasta temp do usuário
	videoPath, err := p.downloadVideoFromMinIO(job.VideoURL, timestamp, userTempDir)
	if err != nil {
		return &ProcessingResult{
			Status:      models.StatusFailed,
			Message:     "Erro ao baixar vídeo do MinIO: " + err.Error(),
			ProcessedAt: time.Now(),
		}
	}
	defer os.Remove(videoPath) // Limpar arquivo temporário

	// Processar o vídeo usando a função real, frames também vão para a temp do usuário
	result := processVideo(videoPath, timestamp, userTempDir)

	processingResult := &ProcessingResult{
		Status:      models.StatusCompleted,
		Message:     result.Message,
		ProcessedAt: time.Now(),
		ZipPath:     result.ZipPath,
		FrameCount:  result.FrameCount,
		Images:      result.Images,
	}

	if result.Status == "failed" {
		processingResult.Status = models.StatusFailed
	}

	log.Printf("✅ Processamento concluído: VideoID=%d, Status=%s", job.VideoID, processingResult.Status)
	return processingResult
}

func (p *Processor) ProcessVideoWithError(job *models.VideoProcessingJob) *ProcessingResult {
	log.Printf("🎬 Iniciando processamento do vídeo (com erro): %s", job.FileName)

	// Simular processamento
	time.Sleep(1 * time.Second)

	result := &ProcessingResult{
		Status:      models.StatusFailed,
		Message:     "Erro durante o processamento do vídeo",
		ProcessedAt: time.Now(),
	}

	log.Printf("❌ Processamento falhou: VideoID=%d", job.VideoID)
	return result
}

func processVideo(videoPath, timestamp, tempDir string) ProcessingResult {
	fmt.Printf("Iniciando processamento: %s\n", videoPath)

	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	framePattern := filepath.Join(tempDir, "frame_%04d.png")

	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vf", "fps=1",
		"-y",
		framePattern,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ProcessingResult{
			Status:  "failed",
			Message: fmt.Sprintf("Erro no ffmpeg: %s\nOutput: %s", err.Error(), string(output)),
		}
	}

	frames, err := filepath.Glob(filepath.Join(tempDir, "*.png"))
	if err != nil || len(frames) == 0 {
		return ProcessingResult{
			Status:  "failed",
			Message: "Nenhum frame foi extraído do vídeo",
		}
	}

	fmt.Printf("📸 Extraídos %d frames\n", len(frames))

	originalFileName := filepath.Base(videoPath)
	originalNameWithoutExt := strings.TrimSuffix(originalFileName, filepath.Ext(originalFileName))
	zipFilename := fmt.Sprintf("%s.zip", originalNameWithoutExt)
	zipPath := filepath.Join("outputs", zipFilename)

	err = createZipFile(frames, zipPath)
	if err != nil {
		return ProcessingResult{
			Status:  "failed",
			Message: "Erro ao criar arquivo ZIP: " + err.Error(),
		}
	}

	fmt.Printf("✅ ZIP criado: %s\n", zipPath)

	imageNames := make([]string, len(frames))
	for i, frame := range frames {
		imageNames[i] = filepath.Base(frame)
	}

	return ProcessingResult{
		Status:     "completed",
		Message:    fmt.Sprintf("Processamento concluído! %d frames extraídos.", len(frames)),
		ZipPath:    zipFilename,
		FrameCount: len(frames),
		Images:     imageNames,
	}
}

func createZipFile(files []string, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		err := addFileToZip(zipWriter, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filepath.Base(filename)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// downloadVideoFromMinIO baixa um vídeo do MinIO para um arquivo local
func (p *Processor) downloadVideoFromMinIO(videoURL, timestamp, userTempDir string) (string, error) {
	if p.minioClient == nil {
		return "", fmt.Errorf("cliente MinIO não configurado")
	}

	parts := strings.Split(videoURL, "/")
	if len(parts) < 4 {
		return "", fmt.Errorf("URL do MinIO inválida: %s", videoURL)
	}
	objectName := strings.Join(parts[4:], "/")

	localPath := filepath.Join(userTempDir, fmt.Sprintf("video_%s.mp4", timestamp))

	err := p.minioClient.DownloadFile(context.Background(), objectName, localPath)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar vídeo do MinIO: %w", err)
	}

	log.Printf("📥 Vídeo baixado do MinIO: %s -> %s", objectName, localPath)
	return localPath, nil
}
