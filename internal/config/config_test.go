package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mymeetily/mymeetily/internal/layout"
)

func TestLoadDefault(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}

	if cfg.ASR.WhisperBinary != layout.WhisperBinary {
		t.Errorf("unexpected whisper binary: %s", cfg.ASR.WhisperBinary)
	}
	if cfg.ASR.ModelPath != layout.WhisperModel {
		t.Errorf("unexpected model path: %s", cfg.ASR.ModelPath)
	}
	if cfg.ASR.Language != "zh" {
		t.Errorf("unexpected asr language: %s", cfg.ASR.Language)
	}
	if cfg.Ollama.Endpoint != "http://127.0.0.1:11434" {
		t.Errorf("unexpected ollama endpoint: %s", cfg.Ollama.Endpoint)
	}
	if cfg.Ollama.Model != "qwen2.5:1.5b-instruct" {
		t.Errorf("unexpected ollama model: %s", cfg.Ollama.Model)
	}
	if cfg.Ollama.Temperature != 0.3 {
		t.Errorf("unexpected temperature: %f", cfg.Ollama.Temperature)
	}
	if cfg.Audio.Backend != "wasapi" {
		t.Errorf("unexpected audio backend: %s", cfg.Audio.Backend)
	}
}

func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	content := "asr:\n" +
		"  whisper_binary: \"/usr/local/bin/whisper-cli\"\n" +
		"  model_path: \"/opt/models/ggml-large-v3.bin\"\n" +
		"  language: \"en\"\n" +
		"ollama:\n" +
		"  endpoint: \"http://127.0.0.1:11434\"\n" +
		"  model: \"qwen2.5:1.5b-instruct\"\n" +
		"  temperature: 0.5\n" +
		"  max_tokens: 2048\n" +
		"audio:\n" +
		"  backend: \"wasapi\"\n" +
		"  sample_rate: 44100\n" +
		"  channels: 2\n" +
		"output:\n" +
		"  output_dir: \"/tmp/output\"\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config from file: %v", err)
	}

	if cfg.ASR.WhisperBinary != "/usr/local/bin/whisper-cli" {
		t.Errorf("unexpected whisper binary: %s", cfg.ASR.WhisperBinary)
	}
	if cfg.ASR.ModelPath != "/opt/models/ggml-large-v3.bin" {
		t.Errorf("unexpected model path: %s", cfg.ASR.ModelPath)
	}
	if cfg.ASR.Language != "en" {
		t.Errorf("unexpected language: %s", cfg.ASR.Language)
	}
	if cfg.Ollama.Endpoint != "http://127.0.0.1:11434" {
		t.Errorf("unexpected ollama endpoint: %s", cfg.Ollama.Endpoint)
	}
	if cfg.Ollama.Model != "qwen2.5:1.5b-instruct" {
		t.Errorf("unexpected ollama model: %s", cfg.Ollama.Model)
	}
	if cfg.Ollama.Temperature != 0.5 {
		t.Errorf("unexpected temperature: %f", cfg.Ollama.Temperature)
	}
	if cfg.Ollama.MaxTokens != 2048 {
		t.Errorf("unexpected max tokens: %d", cfg.Ollama.MaxTokens)
	}
	if cfg.Audio.SampleRate != 44100 {
		t.Errorf("unexpected sample rate: %d", cfg.Audio.SampleRate)
	}
	if cfg.Audio.Backend != "wasapi" {
		t.Errorf("unexpected backend: %s", cfg.Audio.Backend)
	}
}
