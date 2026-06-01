.PHONY: build test lint run clean download-model package

BINARY := mymeetily.exe
MODEL_URL := https://hf-mirror.com/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo-q5_0.bin
MODEL_DIR := llmmodels/whisper
MODEL_FILE := $(MODEL_DIR)/ggml-large-v3-turbo-q5_0.bin
DIST_DIR := dist/mymeetily

build:
	go build -ldflags="-s -w" -o $(BINARY) .

test:
	go test ./internal/... -v -count=1

lint:
	go vet ./...

run:
	go run .

download-model:
	go run . init-model --target all

package: build
	@echo "=== 打包 MyMeetily 分发版 ==="
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)/llmmodels/whisper
	@mkdir -p $(DIST_DIR)/llmmodels/summary
	@mkdir -p $(DIST_DIR)/output
	@mkdir -p $(DIST_DIR)/configs
	@mkdir -p $(DIST_DIR)/llmengine/whispercpp/Release
	@cp $(BINARY) $(DIST_DIR)/
	@cp configs/config.yaml $(DIST_DIR)/configs/
	@cp llmengine/whispercpp/Release/whisper-cli.exe $(DIST_DIR)/llmengine/whispercpp/Release/ 2>/dev/null || echo "  ⚠ whisper-cli.exe 未找到"
	@cp llmengine/whispercpp/Release/whisper.dll $(DIST_DIR)/llmengine/whispercpp/Release/ 2>/dev/null || true
	@cp llmengine/whispercpp/Release/ggml.dll $(DIST_DIR)/llmengine/whispercpp/Release/ 2>/dev/null || true
	@cp llmengine/whispercpp/Release/ggml-base.dll $(DIST_DIR)/llmengine/whispercpp/Release/ 2>/dev/null || true
	@cp llmengine/whispercpp/Release/ggml-cpu.dll $(DIST_DIR)/llmengine/whispercpp/Release/ 2>/dev/null || true
	@cp llmengine/whispercpp/Release/SDL2.dll $(DIST_DIR)/llmengine/whispercpp/Release/ 2>/dev/null || true
	@cp start.bat $(DIST_DIR)/ 2>/dev/null || true
	@echo ""
	@echo "分发包: $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/*.exe
	@echo ""
	@rm -f dist/mymeetily/llmmodels/whisper/*.bin dist/mymeetily/output/*
	@cd dist && tar -czf mymeetily.tar.gz --exclude="llmmodels/whisper/*.bin" --exclude="output/*" -C mymeetily .
	@echo "完成: dist/mymeetily.tar.gz"

clean:
	rm -f $(BINARY) mymeetily mymeetily.exe
	rm -rf output/
	rm -rf dist/
