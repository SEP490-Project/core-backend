package consumer

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// region: ============ Quality Presets Variables =============

// VideoQualityPreset defines the encoding settings for a specific resolution
type VideoQualityPreset struct {
	Resolution string // e.g., "1280:720"
	Bitrate    string // e.g., "2800k"
	MaxRate    string // e.g., "2996k"
	BufSize    string // e.g., "4200k"
}

// qualityPresets maps common resolution labels to their encoding presets
var qualityPresets = map[string]VideoQualityPreset{
	"144p":  {Resolution: "256:144", Bitrate: "200k", MaxRate: "228k", BufSize: "300k"},
	"240p":  {Resolution: "426:240", Bitrate: "400k", MaxRate: "456k", BufSize: "600k"},
	"360p":  {Resolution: "640:360", Bitrate: "800k", MaxRate: "856k", BufSize: "1200k"},
	"480p":  {Resolution: "854:480", Bitrate: "1400k", MaxRate: "1498k", BufSize: "2100k"},
	"720p":  {Resolution: "1280:720", Bitrate: "2800k", MaxRate: "2996k", BufSize: "4200k"},
	"1080p": {Resolution: "1920:1080", Bitrate: "5000k", MaxRate: "5350k", BufSize: "7500k"},
	"1440p": {Resolution: "2560:1440", Bitrate: "8000k", MaxRate: "8560k", BufSize: "12000k"},
}

// endregion

type VideoUploadConsumer struct {
	appRegistry        *application.ApplicationRegistry
	config             *config.AppConfig
	s3StreamingStorage irepository_third_party.S3StreamingStorage
	fileRepository     irepository.GenericRepository[model.File]
	unitOfWork         irepository.UnitOfWork
}

func NewVideoUploadConsumer(appRegistry *application.ApplicationRegistry) *VideoUploadConsumer {
	return &VideoUploadConsumer{
		appRegistry:        appRegistry,
		config:             appRegistry.InfrastructureRegistry.Config,
		s3StreamingStorage: appRegistry.InfrastructureRegistry.ThirdPartyStorage.S3StreamStorage,
		fileRepository:     appRegistry.DatabaseRegistry.FileRepository,
		unitOfWork:         appRegistry.InfrastructureRegistry.UnitOfWork,
	}
}

// Handle xử lý từng message nhận từ RabbitMQ
func (c *VideoUploadConsumer) Handle(ctx context.Context, body []byte) error {
	var msg consumers.VideoUploadMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("❌ Failed to unmarshal VideoUploadMessage", zap.Error(err))
		return fmt.Errorf("failed to unmarshal VideoUploadMessage: %w", err)
	}

	zap.L().Info("📥 Received video upload task",
		zap.Any("message", msg))

	// Get file record from DB
	fileRecord, err := c.fileRepository.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", msg.FileID)
	}, nil)
	if err != nil {
		zap.L().Error("❌ Failed to get file record from DB",
			zap.String("fileID", msg.FileID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get file record: %w", err)
	}

	// Open temp file
	file, err := os.Open(msg.FilePath)
	if err != nil {
		zap.L().Error("❌ Failed to open temp file", zap.String("path", msg.FilePath), zap.Error(err))
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Start transations
	uow := c.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			c.s3StreamingStorage.Delete(ctx, msg.Key)
		}
	}()

	var finalKey string
	if msg.IsHLS {
		// Transcode to HLS
		hlsKey, err := c.transcodeAndUploadHLS(ctx, msg.FilePath, msg.Key, msg.Resolutions, msg.SegmentDuration)
		if err != nil {
			zap.L().Error("❌ Failed to transcode/upload HLS", zap.Error(err))
			return fmt.Errorf("failed to transcode/upload HLS: %w", err)
		}
		finalKey = hlsKey
	} else {
		// Upload raw MP4 to S3
		if err := c.s3StreamingStorage.Put(ctx, msg.Key, file, "video/mp4"); err != nil {
			zap.L().Error("❌ Failed to upload video to S3",
				zap.String("key", msg.Key),
				zap.Error(err),
			)
			return fmt.Errorf("failed to upload to s3: %w", err)
		}
		finalKey = msg.Key
	}

	// Delete temp file
	if err := os.Remove(msg.FilePath); err != nil {
		zap.L().Warn("⚠️ Failed to remove temp file after upload",
			zap.String("path", msg.FilePath),
			zap.Error(err),
		)
	} else {
		zap.L().Info("🧼 Temp file removed", zap.String("path", msg.FilePath))
	}

	// cdnVideoURL := c.s3StreamingStorage.BuildUrl(msg.Key)
	fileRecord.URL = c.s3StreamingStorage.BuildUrl(finalKey)
	fileRecord.Status = enum.FileStatusUploaded
	fileRecord.UploadedAt = utils.PtrOrNil(time.Now())
	if err := c.fileRepository.Update(ctx, fileRecord); err != nil {
		zap.L().Error("❌ Failed to update file record",
			zap.String("fileID", msg.FileID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update file record: %w", err)
	}

	if err := uow.Commit(); err != nil {
		zap.L().Error("❌ Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	zap.L().Info("✅ Video upload completed successfully",
		zap.String("key", msg.Key),
		zap.String("userID", msg.UserID),
	)

	//update database or perform additional actions if needed
	if msg.Action != nil {
		zap.L().Info("Performing post-upload action", zap.String("action", *msg.Action))
		//do something with msg.Action
	}
	return nil
}

func (c *VideoUploadConsumer) transcodeAndUploadHLS(ctx context.Context, inputPath string, originalKey string, resolutions []string, segmentDuration int) (string, error) {
	startTime := time.Now()

	zap.L().Info("Starting HLS transcoding job",
		zap.String("original_key", originalKey),
		zap.Strings("resolutions", resolutions),
		zap.Int("segment_duration", segmentDuration),
	)

	// 1. Setup Directories
	outputDir := inputPath + "_hls"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		zap.L().Error("Failed to create temp directory", zap.Error(err))
		return "", fmt.Errorf("failed to create hls output dir: %w", err)
	}

	// [DEBUG] Context about file system
	zap.L().Debug("Created temporary HLS directory", zap.String("path", outputDir))
	defer func() {
		os.RemoveAll(outputDir)
		zap.L().Debug("Cleaned up temporary HLS directory", zap.String("path", outputDir))
	}()

	if segmentDuration <= 0 {
		segmentDuration = 10
	}

	// 2. Build FFmpeg Arguments
	args, err := buildABRCommand(inputPath, outputDir, resolutions, segmentDuration)
	if err != nil {
		return "", err
	}

	// 3. Execute FFmpeg
	zap.L().Info("Executing FFmpeg transcoding process...")
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Capture output. Using CombinedOutput to get both stdout and stderr
	output, err := cmd.CombinedOutput()

	transcodeDuration := time.Since(startTime)

	if err != nil {
		zap.L().Error("FFmpeg execution failed",
			zap.Error(err),
			zap.String("ffmpeg_output", string(output)),
			zap.Duration("duration_until_fail", transcodeDuration),
		)
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}

	zap.L().Info("FFmpeg transcoding completed successfully",
		zap.Duration("duration", transcodeDuration),
	)
	if c.config.IsDevelopmentDebugging() {
		zap.L().Debug("FFmpeg Output", zap.String("output", string(output)))
	}

	// 4. Upload Logic
	baseKey := strings.TrimSuffix(originalKey, filepath.Ext(originalKey)) + "_hls"
	files, err := os.ReadDir(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to read hls output dir: %w", err)
	}

	zap.L().Info("Starting S3 upload of HLS segments",
		zap.Int("file_count", len(files)),
		zap.String("base_key", baseKey),
	)

	uploadStart := time.Now()
	var batchItems []dtos.BatchUploadItem
	var numSegments, numStreams int64 = 0, 0
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fileName := f.Name()
		localPath := filepath.Join(outputDir, fileName)
		s3Key := fmt.Sprintf("%s/%s", baseKey, fileName)

		// Determine content type
		contentType := "application/octet-stream"
		if strings.HasSuffix(fileName, ".m3u8") {
			contentType = "application/x-mpegURL"
			numStreams++
		} else if strings.HasSuffix(fileName, ".ts") {
			contentType = "video/MP2T"
			numSegments++
		}

		batchItems = append(batchItems, dtos.BatchUploadItem{
			LocalPath:   localPath,
			S3Key:       s3Key,
			ContentType: contentType,
		})
	}

	// 3. Run Batch Upload (using your Utils via the repository)
	zap.L().Info("Starting batch upload",
		zap.Int("files", len(batchItems)),
		zap.Int64("num_streams", numStreams),
		zap.Int64("num_segments", numSegments))

	if err := c.s3StreamingStorage.BatchPut(ctx, batchItems); err != nil {
		return "", fmt.Errorf("failed to batch upload HLS files: %w", err)
	}

	totalDuration := time.Since(startTime)

	zap.L().Info("HLS Transcode and Upload finished successfully",
		zap.Duration("upload_duration", time.Since(uploadStart)),
		zap.Duration("total_duration", totalDuration),
		zap.String("master_playlist", fmt.Sprintf("%s/master.m3u8", baseKey)),
	)

	return fmt.Sprintf("%s/master.m3u8", baseKey), nil
}

func buildABRCommand(inputPath, outputDir string, resolutions []string, segDuration int) ([]string, error) {
	zap.L().Debug("Building FFmpeg ABR arguments",
		zap.Strings("requested_resolutions", resolutions),
	)

	args := []string{
		"-hide_banner", "-y",
		"-i", inputPath,
	}

	if len(resolutions) == 0 {
		resolutions = []string{"720p"}
	}

	var filterComplex strings.Builder
	var varStreamMap strings.Builder

	// 1. Create the Split Filter
	// [0:v]split=5[v0][v1][v2][v3][v4]
	filterComplex.WriteString(fmt.Sprintf("[0:v]split=%d", len(resolutions)))
	for i := 0; i < len(resolutions); i++ {
		filterComplex.WriteString(fmt.Sprintf("[v%d]", i))
	}
	filterComplex.WriteString(";")

	// 2. Split audio: [0:a]asplit=N[a0][a1]...
	// This creates a dedicated audio stream for each resolution.
	filterComplex.WriteString(fmt.Sprintf("[0:a]asplit=%d", len(resolutions)))
	for i := 0; i < len(resolutions); i++ {
		filterComplex.WriteString(fmt.Sprintf("[a%d]", i))
	}
	filterComplex.WriteString(";")

	// 3. Create Scale Filters
	for i, resName := range resolutions {
		preset, ok := qualityPresets[resName]
		if !ok {
			zap.L().Warn("Unknown resolution requested, defaulting to 720p preset", zap.String("requested", resName))
			preset = qualityPresets["720p"]
		}
		filterComplex.WriteString(fmt.Sprintf("[v%d]scale=%s[v%dout]", i, preset.Resolution, i))
		if i < len(resolutions)-1 {
			filterComplex.WriteString(";")
		}
	}
	args = append(args, "-filter_complex", filterComplex.String())

	// 4. Encoding Settings & Stream Mapping
	for i, resName := range resolutions {
		preset, ok := qualityPresets[resName]
		if !ok {
			preset = qualityPresets["720p"]
		}

		zap.L().Debug("Applying quality preset",
			zap.String("resolution_name", resName),
			zap.String("video_bitrate", preset.Bitrate),
			zap.String("resolution_size", preset.Resolution),
		)

		args = append(args,
			"-map", fmt.Sprintf("[v%dout]", i),
			fmt.Sprintf("-c:v:%d", i), "libx264",
			fmt.Sprintf("-b:v:%d", i), preset.Bitrate,
			fmt.Sprintf("-maxrate:v:%d", i), preset.MaxRate,
			fmt.Sprintf("-bufsize:v:%d", i), preset.BufSize,
			"-g", fmt.Sprintf("%d", segDuration*30),
			"-keyint_min", fmt.Sprintf("%d", segDuration*30),
			"-sc_threshold", "0",
		)

		args = append(args,
			"-map", fmt.Sprintf("[a%d]", i),
			fmt.Sprintf("-c:a:%d", i), "aac",
			fmt.Sprintf("-b:a:%d", i), "128k",
			"-ac", "2",
		)

		if i > 0 {
			varStreamMap.WriteString(" ")
		}
		varStreamMap.WriteString(fmt.Sprintf("v:%d,a:%d", i, i))
	}

	args = append(args,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", segDuration),
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-master_pl_name", "master.m3u8",
		"-var_stream_map", varStreamMap.String(),
		"-hls_segment_filename", fmt.Sprintf("%s/stream_%%v_data%%03d.ts", outputDir),
		fmt.Sprintf("%s/stream_%%v.m3u8", outputDir),
	)

	// Log the full generated command for reproduction
	zap.L().Debug("FFmpeg command constructed", zap.String("full_command", "ffmpeg "+strings.Join(args, " ")))

	return args, nil
}
