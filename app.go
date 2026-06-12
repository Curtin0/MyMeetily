package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mymeetily/mymeetily/internal/agent"
	"github.com/mymeetily/mymeetily/internal/audio"
	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/workflow"
)

// =============================================================================
// App — 应用生命周期管理
// =============================================================================

type App struct {
	ctx             context.Context
	cfg             *config.Config
	appService      *AppService
	deviceService   *DeviceService
	recordService   *RecordService
	pipelineService *PipelineService
	resultService   *ResultService
}

func NewApp() *App {
	cfg, err := config.Load("")
	if err != nil {
		// Fall back to defaults — config.Load already populates defaults on error
		cfg, _ = config.Load("")
	}

	a := &App{cfg: cfg}
	a.appService = &AppService{app: a}
	a.deviceService = &DeviceService{app: a}
	a.recordService = &RecordService{app: a}
	a.pipelineService = &PipelineService{app: a}
	a.resultService = &ResultService{app: a}
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.appService.ctx = ctx
	a.deviceService.ctx = ctx
	a.recordService.ctx = ctx
	a.pipelineService.ctx = ctx
	a.resultService.ctx = ctx

	// Run startup pre-flight checks in background
	go a.runStartupChecks()
}

func (a *App) shutdown(ctx context.Context) {
	if a.recordService != nil && a.recordService.IsRecording() {
		a.recordService.StopRecording()
	}
}

func (a *App) runStartupChecks() {
	// Emit status update
	runtime.EventsEmit(a.ctx, "app:status", map[string]interface{}{
		"phase":   "checking",
		"message": "正在检查依赖...",
	})

	// Check whisper model file
	whisperModelOK := true
	whisperModelMsg := "语音模型已就绪"
	if _, err := os.Stat(a.cfg.ASR.ModelPath); os.IsNotExist(err) {
		whisperModelOK = false
		whisperModelMsg = "语音模型未找到，请运行 init-model 命令下载模型"
	}

	// Check Ollama
	summaryEnabled := false
	ollamaOK := false
	ollamaMsg := "Ollama 服务未连接"
	modelName := a.cfg.Ollama.Model

	startupStatus, err := workflow.DetectSummaryAvailability(a.ctx, a.cfg,
		"./llmmodels/summary/qwen2.5-1.5b-instruct-q4_k_m.gguf",
		"https://hf-mirror.com/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf",
	)
	if err == nil {
		ollamaOK = true
		summaryEnabled = startupStatus.SummaryEnabled
		if summaryEnabled {
			ollamaMsg = fmt.Sprintf("Ollama 已就绪 (%s)", modelName)
		} else {
			ollamaMsg = startupStatus.SummaryMessage
		}
	} else {
		ollamaMsg = fmt.Sprintf("Ollama 未连接: %v", err)
	}

	// Emit dependency check results
	runtime.EventsEmit(a.ctx, "app:dependency", map[string]interface{}{
		"whisperBinary": map[string]interface{}{
			"ok":      true,
			"message": "引擎已就绪",
		},
		"whisperModel": map[string]interface{}{
			"ok":      whisperModelOK,
			"message": whisperModelMsg,
		},
		"ollama": map[string]interface{}{
			"ok":      ollamaOK,
			"message": ollamaMsg,
		},
		"summaryEnabled": summaryEnabled,
	})

	// Emit ready status
	runtime.EventsEmit(a.ctx, "app:status", map[string]interface{}{
		"phase":          "ready",
		"message":        "就绪",
		"summaryEnabled": summaryEnabled,
	})

	// Emit app info
	runtime.EventsEmit(a.ctx, "app:info", map[string]interface{}{
		"engine":       "whisper.cpp + Ollama",
		"whisperModel": filepath.Base(a.cfg.ASR.ModelPath),
		"llmModel":     a.cfg.Ollama.Model,
	})
}

// =============================================================================
// AppService — 应用信息和系统操作 (暴露给前端)
// =============================================================================

type AppService struct {
	ctx context.Context
	app *App
}

type AppInfo struct {
	Engine       string `json:"engine"`
	WhisperModel string `json:"whisperModel"`
	LLMModel     string `json:"llmModel"`
}

func (s *AppService) GetAppInfo() AppInfo {
	return AppInfo{
		Engine:       "whisper.cpp + Ollama",
		WhisperModel: filepath.Base(s.app.cfg.ASR.ModelPath),
		LLMModel:     s.app.cfg.Ollama.Model,
	}
}

func (s *AppService) OpenOutputFolder(path string) {
	dir := filepath.Dir(path)
	runtime.BrowserOpenURL(s.ctx, "file:///"+filepath.ToSlash(dir))
}

func (s *AppService) CopyToClipboard(text string) error {
	return runtime.ClipboardSetText(s.ctx, text)
}

func (s *AppService) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "选择音频文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Audio Files (*.mp3;*.wav;*.m4a;*.flac;*.ogg;*.wma;*.aac)",
				Pattern:     "*.mp3;*.wav;*.m4a;*.flac;*.ogg;*.wma;*.aac",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	})
}

// =============================================================================
// DeviceService — 音频设备枚举 (暴露给前端)
// =============================================================================

type DeviceService struct {
	ctx context.Context
	app *App
}

type DeviceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (s *DeviceService) ListMicDevices() ([]DeviceInfo, error) {
	devices, err := audio.ListCaptureDevices(audio.BackendWASAPI)
	if err != nil {
		return nil, err
	}
	result := make([]DeviceInfo, 0, len(devices))
	for _, d := range devices {
		result = append(result, DeviceInfo{
			ID:   d,
			Name: d,
			Type: "mic",
		})
	}
	return result, nil
}

func (s *DeviceService) ListSpeakerDevices() ([]DeviceInfo, error) {
	devices, err := audio.ListLoopbackDevices(audio.BackendWASAPI)
	if err != nil {
		return nil, err
	}
	result := make([]DeviceInfo, 0, len(devices))
	for _, d := range devices {
		result = append(result, DeviceInfo{
			ID:   d,
			Name: d,
			Type: "speaker",
		})
	}
	return result, nil
}

// =============================================================================
// RecordService — 录音控制和实时数据推送 (暴露给前端)
// =============================================================================

type RecordService struct {
	ctx         context.Context
	cancel      context.CancelFunc
	app         *App
	mu          sync.Mutex
	recorder    audio.Recorder
	transcriber *audio.LiveTranscriber
	running     bool
	outputPath  string
	startTime   time.Time
}

func (s *RecordService) StartRecording(micDevice, speakerDevice string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("已经在录音中")
	}

	outputDir := s.app.cfg.Output.OutputDir
	outputPath := audio.BuildRecordingPath(outputDir, time.Now())

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// Create live transcriber
	transcriber := audio.NewLiveTranscriber(audio.LiveTranscribeConfig{
		WhisperBinary: s.app.cfg.ASR.WhisperBinary,
		ModelPath:     s.app.cfg.ASR.ModelPath,
		Language:      s.app.cfg.ASR.Language,
	})

	// Create recorder config
	cfg := audio.RecorderConfig{
		MicDevice:     micDevice,
		SpeakerDevice: speakerDevice,
		OutputPath:    outputPath,
		OnAudioData:   transcriber.Feed,
	}

	recorder, err := audio.NewRecorder(cfg)
	if err != nil {
		transcriber.Close()
		return fmt.Errorf("创建录音器失败: %w", err)
	}

	if err := recorder.Start(); err != nil {
		transcriber.Close()
		return fmt.Errorf("启动录音失败: %w", err)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	s.cancel = cancel
	s.recorder = recorder
	s.transcriber = transcriber
	s.running = true
	s.outputPath = outputPath
	s.startTime = time.Now()

	// Emit started event
	runtime.EventsEmit(s.ctx, "record:started", map[string]interface{}{
		"outputPath": outputPath,
	})

	// Start background goroutine for real-time data push
	go s.pushLoop(ctx)

	return nil
}

func (s *RecordService) StopRecording() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return "", fmt.Errorf("未在录音")
	}

	// Cancel push loop
	if s.cancel != nil {
		s.cancel()
	}

	// Stop recorder
	if s.recorder != nil {
		s.recorder.Stop()
	}

	// Flush and close transcriber
	if s.transcriber != nil {
		s.transcriber.Close()
	}

	outputPath := s.outputPath
	s.running = false
	s.recorder = nil
	s.transcriber = nil

	// Emit stopped event
	runtime.EventsEmit(s.ctx, "record:stopped", map[string]interface{}{
		"outputPath": outputPath,
		"duration":   time.Since(s.startTime).Seconds(),
	})

	return outputPath, nil
}

func (s *RecordService) IsRecording() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *RecordService) GetElapsed() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return 0
	}
	return time.Since(s.startTime).Seconds()
}

// pushLoop runs in the background, pushing peak level and transcript data
func (s *RecordService) pushLoop(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.running || s.recorder == nil {
				s.mu.Unlock()
				return
			}

			// Push peak level
			peakLevel := s.recorder.PeakLevel()
			elapsed := time.Since(s.startTime).Seconds()
			runtime.EventsEmit(s.ctx, "record:peaklevel", map[string]interface{}{
				"level":   peakLevel,
				"elapsed": elapsed,
			})

			// Drain transcript results
			if s.transcriber != nil {
				for {
					select {
					case text, ok := <-s.transcriber.Results:
						if !ok {
							s.mu.Unlock()
							return
						}
						if text != "" && text != "__pending__" {
							runtime.EventsEmit(s.ctx, "record:transcript", map[string]interface{}{
								"text": text,
							})
						}
					default:
						goto doneTranscript
					}
				}
			doneTranscript:
			}

			s.mu.Unlock()
		}
	}
}

// =============================================================================
// PipelineService — 处理管线调用 (暴露给前端)
// =============================================================================

type PipelineService struct {
	ctx        context.Context
	app        *App
	mu         sync.Mutex
	currentCtx context.Context
	cancel     context.CancelFunc
	running    bool
}

func (s *PipelineService) RunPipeline(audioPath string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("已有处理管线在运行")
	}
	s.running = true
	ctx, cancel := context.WithCancel(s.ctx)
	s.currentCtx = ctx
	s.cancel = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	// Create progress callback that emits Wails events
	progressFn := func(stage string) {
		runtime.EventsEmit(s.ctx, "pipeline:progress", map[string]interface{}{
			"stage": stage,
		})
	}

	state, err := workflow.RunMeetingPipeline(
		ctx,
		s.app.cfg,
		audioPath,
		s.app.cfg.ASR.Language,
		progressFn,
	)

	if err != nil {
		runtime.EventsEmit(s.ctx, "pipeline:error", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	// Store result in result service
	s.app.resultService.SetState(state)

	// Emit done with serialized state
	runtime.EventsEmit(s.ctx, "pipeline:done", stateToMap(state))

	return nil
}

func (s *PipelineService) RetrySummary() error {
	if s.app.resultService.state == nil {
		return fmt.Errorf("没有可重试的会议记录")
	}

	runtime.EventsEmit(s.ctx, "pipeline:progress", map[string]interface{}{
		"stage": "LLM 整理",
	})

	newState, err := agent.RetrySummary(s.ctx, s.app.cfg, s.app.resultService.state)
	if err != nil {
		runtime.EventsEmit(s.ctx, "pipeline:error", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	s.app.resultService.SetState(newState)
	runtime.EventsEmit(s.ctx, "pipeline:done", stateToMap(newState))
	return nil
}

func (s *PipelineService) ImportAudioFile(filePath string) error {
	return s.RunPipeline(filePath)
}

// =============================================================================
// ResultService — 结果数据访问 (暴露给前端)
// =============================================================================

type ResultService struct {
	ctx   context.Context
	app   *App
	mu    sync.Mutex
	state *agent.MeetingState
}

func (s *ResultService) SetState(state *agent.MeetingState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *ResultService) GetState() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return nil
	}
	return stateToMap(s.state)
}

// =============================================================================
// Helpers
// =============================================================================

func stateToMap(state *agent.MeetingState) map[string]interface{} {
	if state == nil {
		return nil
	}

	segments := make([]map[string]interface{}, 0, len(state.Segments))
	for _, seg := range state.Segments {
		segments = append(segments, map[string]interface{}{
			"start":      seg.Start,
			"end":        seg.End,
			"text":       seg.Text,
			"confidence": seg.Confidence,
		})
	}

	return map[string]interface{}{
		"audioFilePath":  state.AudioFilePath,
		"language":       state.Language,
		"rawTranscript":  state.RawTranscript,
		"meetingNotes":   state.MeetingNotes,
		"summaryContent": state.SummaryContent,
		"summaryError":   state.SummaryError,
		"summaryEnabled": state.SummaryEnabled,
		"audioDuration":  state.AudioDuration,
		"outputPath":     state.HTMLOutputPath,
		"htmlOutputPath": state.HTMLOutputPath,
		"transcriptPath": state.TranscriptPath,
		"markdownPath":   state.MarkdownPath,
		"segments":       segments,
		"error":          state.Error,
	}
}
