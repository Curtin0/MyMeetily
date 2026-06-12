# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / Test / Lint

```bash
wails build                                      # production build (or: make build)
wails dev                                        # dev mode with hot-reload (or: make dev)
go run . init-engine                             # CLI: download whisper.cpp engine
go run . init-model                              # CLI: download models
go test ./internal/... -v -count=1               # test (or: make test)
go vet ./...                                     # lint (or: make lint)
make package                                     # build + package into dist/
make clean                                       # remove build artifacts
```

Single-package test example: `go test ./internal/config/... -v -run TestLoad`

## UI Design Guidelines

- **Color palette**: gray-white minimalist, **no more than 3 colors**
  - Text / Headings: `#1e293b` / `#334155` / `#475569` (dark slate grays)
  - Background / Surfaces: `#ffffff` / `#f8fafc` / `#f1f5f9` (white / near-white)
  - Accent / Interactive: `#3b82f6` (steel blue) — buttons, links, focus rings, active states only
- Borders / Dividers: `#e2e8f0` (light gray, not counted as a color)
- Muted / Secondary text: `#64748b` / `#94a3b8` (gray, not counted as a color)
- Never introduce additional colors (no green, orange, red, purple, etc.) beyond status indicators
- Status colors (dot indicators only): green `#10b981` for OK, amber `#f59e0b` for warning, red `#ef4444` for recording — keep these minimal
- All CSS uses inline styles via `React.CSSProperties` (no separate CSS files per component)
- Font: system native sans-serif stack

## Subcommands

- Default (no args) — launch the Wails v2 desktop GUI
- `go run . init-engine` — download and extract whisper.cpp CUDA engine binaries
- `go run . init-model` — download whisper model (GGUF) and pull/setup the Ollama LLM model

## Architecture

MyMeetily is a Windows desktop GUI app (Wails v2 + React/TypeScript) that records system+mic audio, transcribes it offline with whisper.cpp, and produces structured Chinese meeting minutes via Ollama.

### Startup flow

`main.go` → routes to either CLI subcommand or Wails GUI:
- CLI mode: `cmd/app.go:Run()` handles `init-engine` / `init-model`
- GUI mode: `wails.Run()` with embedded `App` struct → `app.go:startup()` runs pre-flight checks, emits events to frontend

### Frontend (React + TypeScript)

`frontend/src/` — React single-page dashboard with phase-based rendering:

```
<App>
  <TitleBar />          // App title, status indicator, summary mode badge
  <MainLayout>
    <Sidebar>
      <DevicePanel />   // Mic/speaker selection dropdowns
      <ControlPanel />  // Start/stop recording button + timer
      <StatusPanel />   // Dependency status (whisper, Ollama)
    </Sidebar>
    <ContentArea>
      idle | <LiveView> | <ProgressView> | <ResultView>
    </ContentArea>
  </MainLayout>
</App>
```

State management: `useReducer` + Context in `hooks/useAppState.ts`.
Wails events: `hooks/useWailsEvents.ts` subscribes to backend events and dispatches to reducer.

### Wails bound services (Go → JS bridge)

`app.go` — all Wails-bound service structs in the main package:

| Service | Methods | Purpose |
|---------|---------|---------|
| `AppService` | `GetAppInfo`, `OpenOutputFolder`, `CopyToClipboard` | App metadata + system ops |
| `DeviceService` | `ListMicDevices`, `ListSpeakerDevices` | WASAPI device enumeration |
| `RecordService` | `StartRecording`, `StopRecording`, `IsRecording`, `GetElapsed` | Recording lifecycle |
| `PipelineService` | `RunPipeline`, `RetrySummary`, `ImportAudioFile` | Processing pipeline |
| `ResultService` | `GetState` | Meeting state accessor |

Real-time data flows via Wails Events:
- `record:peaklevel` (10Hz) — audio level meter
- `record:transcript` (per sentence) — live transcription
- `pipeline:progress` (4 stages) — processing progress
- `pipeline:done` / `pipeline:error` — completion/failure

### Processing pipeline (Eino agent graph)

`internal/agent/graph.go` builds a 4-node linear DAG using CloudWeGo Eino:

```
AudioPreprocess → ASR → Summarize → FormatOutput
```

Each node is a method on `agent.Nodes` (`internal/agent/nodes_shared.go`):
- **AudioPreprocess** — passes the recorded audio file through (whisper.cpp handles formats natively)
- **ASR** — calls `asr.Client.Transcribe()` which shells out to `whisper-cli -oj`
- **Summarize** — sends the raw transcript to Ollama HTTP API with a Chinese meeting-minutes system prompt (`internal/ollama/prompt.go`)
- **FormatOutput** — writes 4 files: `.html` (styled meeting notes), `.md` (markdown), `.txt` (raw transcript), and `.wav` (original recording)

All pipeline state flows through `agent.MeetingState` (`internal/agent/state.go`). The GUI layer sets `ProgressFn` on the state to receive stage progress updates via Wails events.

### Real-time transcription during recording

`internal/audio/livetranscribe.go` — runs alongside the WASAPI recorder:
1. `Feed()` receives raw PCM chunks from the capture loop
2. `vadLoop()` runs energy-based VAD (RMS threshold), cuts sentences on 1s silence, writes temp WAV files
3. Each sentence WAV is transcribed by `whisper-cli` in its own goroutine
4. Results are sent through a buffered channel consumed by `RecordService.pushLoop()` which emits Wails events

### Audio capture

`internal/audio/` — WASAPI loopback + mic capture. Platform-specific files:
- `wasapi_windows.go` / `devices_wasapi_windows.go` — real implementations using go-wca
- `wasapi_other.go` / `devices_wasapi_other.go` — stubs returning errors on non-Windows

The `Recorder` interface (`recorder.go`): `Start()/Stop()/Elapsed()/PeakLevel()`.

### Key external dependencies

- **Wails v2** — desktop GUI framework (Go backend + WebView2 frontend)
- **React + TypeScript** — frontend UI (Vite build system)
- **whisper.cpp** — external binary at `llmengine/whispercpp/Release/whisper-cli.exe`. Large-v3-turbo Q5_0 GGUF model at `llmmodels/whisper/`. Both downloaded by `init-engine` / `init-model` subcommands.
- **Ollama** — must be running as a local HTTP service (default `http://127.0.0.1:11434`). Default model: `qwen2.5:1.5b-instruct`.

### Configuration

`configs/config.yaml` loaded by `internal/config/config.go` using Viper. Falls back to defaults defined in `internal/layout/paths.go`. Sections: `asr`, `ollama`, `audio`, `output`.

### Output structure

Each session creates `output/<timestamp>/` with:
- `live_<timestamp>.wav` — original recording
- `live_<timestamp>_转写.txt` — raw ASR transcript
- `live_<timestamp>_纪要.html` — styled meeting minutes (Chinese)
- `live_<timestamp>_纪要.md` — markdown version
