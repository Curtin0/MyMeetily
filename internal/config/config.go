package config

import (
	"fmt"

	"github.com/mymeetily/mymeetily/internal/layout"
	"github.com/spf13/viper"
)

type Config struct {
	ASR    ASRConfig    `mapstructure:"asr"`
	Ollama OllamaConfig `mapstructure:"ollama"`
	Audio  AudioConfig  `mapstructure:"audio"`
	Output OutputConfig `mapstructure:"output"`
}

type ASRConfig struct {
	WhisperBinary string `mapstructure:"whisper_binary"`
	ModelPath     string `mapstructure:"model_path"`
	Language      string `mapstructure:"language"`
}

type OllamaConfig struct {
	Endpoint    string  `mapstructure:"endpoint"`
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

type AudioConfig struct {
	Backend    string `mapstructure:"backend"`
	SampleRate int    `mapstructure:"sample_rate"`
	Channels   int    `mapstructure:"channels"`
}

type OutputConfig struct {
	OutputDir string `mapstructure:"output_dir"`
}

func Load(cfgPath string) (*Config, error) {
	v := viper.New()

	v.SetDefault("asr.whisper_binary", layout.WhisperBinary)
	v.SetDefault("asr.model_path", layout.WhisperModel)
	v.SetDefault("asr.language", "zh")
	v.SetDefault("ollama.endpoint", "http://127.0.0.1:11434")
	v.SetDefault("ollama.model", "qwen2.5:1.5b-instruct")
	v.SetDefault("ollama.temperature", 0.3)
	v.SetDefault("ollama.max_tokens", 4096)
	v.SetDefault("audio.backend", "wasapi")
	v.SetDefault("audio.sample_rate", 16000)
	v.SetDefault("audio.channels", 1)
	v.SetDefault("output.output_dir", "./output")

	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
