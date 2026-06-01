# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / Test / Lint

```bash
go build -ldflags="-s -w" -o mymeetily.exe .   # build (or: make build)
go run .                                        # run interactive TUI (or: make run)
go test ./internal/... -v -count=1              # test (or: make test)
go vet ./...                                    # lint (or: make lint)
make package                                    # build + package into dist/
make clean                                      # remove build artifacts
```

Single-package test example: `go test ./internal/config/... -v -run TestLoad`

## Subcommands

- `go run .` — launch the interactive Bubble Tea TUI (default)
- `go run . init-engine` — download and extract whisper.cpp CUDA engine binaries
- `go run . init-model` — download whisper model (GGUF) and pull/setup the Ollama LLM model

## Architecture

MyMeetily is a Windows desktop TUI app that records system+mic audio, transcribes it offline with whisper.cpp, and produces structured Chinese meeting minutes via Ollama.

### Startup flow

`main.go` → `cmd/mymeetily/app.go:Run()` → loads config from YAML, runs pre-flight checks (whisper model exists, Ollama is reachable and has the configured model), then starts a `tea.Program` with the root `tui.Model`.

### TUI screen flow (Bubble Tea)

```
Splash → Menu → MicSelect → Record → Progress → Result
                   ↘ FilePicker (import existing audio) → Progress → Result
```

Each screen is a sub-model on `tui.Model` (file: `internal/tui/model.go`). Screens are dispatched via `m.screen` switch in `Update()`/`View()`. `esc` returns to Menu from most screens. The model also holds a `Check` screen run before first recording to verify Ollama readiness.

### Processing pipeline (Eino agent graph)

`internal/agent/graph.go` builds a 4-node linear DAG using CloudWeGo Eino:

```
AudioPreprocess → ASR → Summarize → FormatOutput
```

Each node is a method on `agent.Nodes` (`internal/agent/nodes.go`):
- **AudioPreprocess** — passes the recorded audio file through (whisper.cpp handles formats natively)
- **ASR** — calls `asr.Client.Transcribe()` which shells out to `whisper-cli -oj`
- **Summarize** — sends the raw transcript to Ollama HTTP API with a Chinese meeting-minutes system prompt (`internal/ollama/prompt.go`)
- **FormatOutput** — writes 3 files: `.html` (styled meeting notes), `.txt` (raw transcript), and `.wav` (original recording)

All pipeline state flows through `agent.MeetingState` (`internal/agent/state.go`). The TUI layer sets `ProgressFn` on the state to receive stage progress updates via a buffered channel.

### Real-time transcription during recording

`internal/audio/livetranscribe.go` — runs alongside the WASAPI recorder:
1. `Feed()` receives raw PCM chunks from the capture loop
2. `vadLoop()` runs energy-based VAD (RMS threshold), cuts sentences on 1s silence, writes temp WAV files
3. Each sentence WAV is transcribed by `whisper-cli` in its own goroutine
4. Results are sent through a buffered channel consumed by `RecordModel.drainTranscript()` on each tick

### Audio capture

`internal/audio/` — WASAPI loopback + mic capture. Platform-specific files:
- `wasapi_windows.go` / `devices_wasapi_windows.go` — real implementations using go-wca
- `wasapi_other.go` / `devices_wasapi_other.go` — stubs returning errors on non-Windows

The `Recorder` interface (`recorder.go`) is intentionally simple: `Start()/Stop()/Elapsed()/PeakLevel()`.

### Key external dependencies

- **whisper.cpp** — external binary at `llmengine/whispercpp/Release/whisper-cli.exe`. Large-v3-turbo Q5_0 GGUF model at `llmmodels/whisper/`. Both downloaded by `init-engine` / `init-model` subcommands.
- **Ollama** — must be running as a local HTTP service (default `http://127.0.0.1:11434`). Default model: `qwen2.5:1.5b-instruct`. The app can auto-import a GGUF into Ollama via `ollama create`.

### Configuration

`configs/config.yaml` loaded by `internal/config/config.go` using Viper. Falls back to defaults defined in `internal/layout/paths.go`. Sections: `asr`, `ollama`, `audio`, `output`.

### Output structure

Each session creates `output/<timestamp>/` with:
- `live_<timestamp>.wav` — original recording
- `live_<timestamp>_转写.txt` — raw ASR transcript
- `live_<timestamp>_纪要.html` — styled meeting minutes (Chinese)
