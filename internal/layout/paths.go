package layout

import "path/filepath"

const (
	LLMEngineDir        = "llmengine"
	LLMModelsDir        = "llmmodels"
	WhisperEngineRelDir = "llmengine/whispercpp/Release"
	WhisperBinary       = "llmengine/whispercpp/Release/whisper-cli"
	WhisperModel        = "./llmmodels/whisper/ggml-large-v3-turbo-q5_0.bin"
	ConfigFile          = "configs/config.yaml"
	GoModFile           = "go.mod"
)

func WhisperEngineDir(root string) string {
	return filepath.Join(root, "llmengine", "whispercpp", "Release")
}

func WhisperBinaryPath(root string) string {
	return filepath.Join(root, "llmengine", "whispercpp", "Release", "whisper-cli.exe")
}

func WhisperModelPath(root string) string {
	return filepath.Join(root, "llmmodels", "whisper", "ggml-large-v3-turbo-q5_0.bin")
}

func ConfigPath(root string) string {
	return filepath.Join(root, "configs", "config.yaml")
}

func GoModPath(root string) string {
	return filepath.Join(root, GoModFile)
}