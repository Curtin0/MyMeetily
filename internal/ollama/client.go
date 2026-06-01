package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const defaultEndpoint = "http://127.0.0.1:11434"

type Client struct {
	endpoint    string
	model       string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
}

func NewClient(endpoint, model string, temperature float64, maxTokens int) *Client {
	trimmedEndpoint := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if trimmedEndpoint == "" {
		trimmedEndpoint = defaultEndpoint
	}
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" {
		trimmedModel = "qwen2.5:1.5b-instruct"
	}
	return &Client{
		endpoint:    trimmedEndpoint,
		model:       trimmedModel,
		temperature: temperature,
		maxTokens:   maxTokens,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Summarize(ctx context.Context, transcript string) (string, error) {
	if err := c.Ping(ctx); err != nil {
		return "", err
	}
	available, err := c.HasModel(ctx)
	if err != nil {
		return "", err
	}
	if !available {
		return "", fmt.Errorf("Ollama 模型 %s 尚未准备好，请先运行 go run . init-model", c.model)
	}

	reqBody := chatRequest{
		Model:  c.model,
		Stream: false,
		Messages: []chatMessage{
			{Role: "system", Content: MeetingSummarySystemPrompt},
			{Role: "user", Content: transcript},
		},
		Options: map[string]any{
			"temperature": c.temperature,
			"num_predict": c.maxTokens,
		},
	}

	content, err := c.postChat(ctx, reqBody)
	if err != nil {
		return "", err
	}
	return content, nil
}

// Ping verifies that the configured Ollama service is reachable.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinURL(c.endpoint, "api/version"), nil)
	if err != nil {
		return fmt.Errorf("创建 Ollama 连接请求失败: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接到本机 Ollama 服务 %s: %w", c.endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Ollama 服务返回异常状态: %s", resp.Status)
	}
	return nil
}

// HasModel reports whether the configured Ollama model already exists locally.
func (c *Client) HasModel(ctx context.Context) (bool, error) {
	available, err := c.hasModel(ctx)
	if err != nil {
		return false, err
	}
	if available {
		return true, nil
	}
	return false, nil
}

// EnsureModelAvailable makes sure the configured Ollama model exists locally.
// If the model is missing, it can download a GGUF source file and import it
// into the local Ollama model store.
func (c *Client) EnsureModelAvailable(ctx context.Context, localGGUFPath, downloadURL string, force bool) error {
	if !force {
		available, err := c.HasModel(ctx)
		if err != nil {
			return err
		}
		if available {
			return nil
		}
	}

	ggufPath, err := c.ensureLocalGGUF(localGGUFPath, downloadURL, force)
	if err != nil {
		return err
	}

	return c.createModelFromGGUF(ctx, ggufPath)
}

func (c *Client) hasModel(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinURL(c.endpoint, "api/tags"), nil)
	if err != nil {
		return false, fmt.Errorf("创建 Ollama 模型检查请求失败: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("获取 Ollama 模型列表失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return false, fmt.Errorf("Ollama 模型列表返回异常状态: %s", resp.Status)
	}

	var payload tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, fmt.Errorf("解析 Ollama 模型列表失败: %w", err)
	}
	for _, model := range payload.Models {
		if strings.EqualFold(strings.TrimSpace(model.Name), c.model) {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) ensureLocalGGUF(localGGUFPath, downloadURL string, force bool) (string, error) {
	localGGUFPath = strings.TrimSpace(localGGUFPath)
	if localGGUFPath == "" {
		return "", fmt.Errorf("本地 GGUF 路径不能为空")
	}

	if !force && fileExists(localGGUFPath) {
		return filepath.Abs(localGGUFPath)
	}

	source := strings.TrimSpace(downloadURL)
	if source == "" {
		return "", fmt.Errorf("GGUF 下载地址不能为空")
	}

	if err := os.MkdirAll(filepath.Dir(localGGUFPath), 0o755); err != nil {
		return "", fmt.Errorf("创建 GGUF 目录失败: %w", err)
	}
	if err := downloadFile(localGGUFPath, source); err != nil {
		return "", fmt.Errorf("下载 GGUF 模型失败: %w", err)
	}

	return filepath.Abs(localGGUFPath)
}

func (c *Client) createModelFromGGUF(ctx context.Context, ggufPath string) error {
	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		return fmt.Errorf("未找到 ollama 命令，请先安装 Ollama: https://ollama.com/download/windows")
	}

	tempDir, err := os.MkdirTemp("", "mymeetily-ollama-create-*")
	if err != nil {
		return fmt.Errorf("创建 Ollama 临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	modelFilePath := filepath.Join(tempDir, "model.gguf")
	if err := copyFile(ggufPath, modelFilePath); err != nil {
		return fmt.Errorf("复制 GGUF 到临时目录失败: %w", err)
	}

	modelfilePath := filepath.Join(tempDir, "Modelfile")
	modelfile := []byte("FROM ./model.gguf\n")
	if err := os.WriteFile(modelfilePath, modelfile, 0o644); err != nil {
		return fmt.Errorf("写入 Ollama Modelfile 失败: %w", err)
	}

	cmd := exec.CommandContext(ctx, ollamaPath, "create", c.model, "-f", modelfilePath)
	if c.endpoint != defaultEndpoint {
		cmd.Env = append(os.Environ(), "OLLAMA_HOST="+c.endpoint)
	}
	cmd.Dir = tempDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("ollama create 失败: %w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("ollama create 失败: %w", err)
	}
	return nil
}

func (c *Client) postChat(ctx context.Context, body chatRequest) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("构造 Ollama 请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(c.endpoint, "api/chat"), bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("创建 Ollama 聊天请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 Ollama 聊天接口失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("Ollama 聊天接口返回异常状态: %s", resp.Status)
	}

	var payload chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("解析 Ollama 响应失败: %w", err)
	}
	if strings.TrimSpace(payload.Error) != "" {
		return "", fmt.Errorf("%s", strings.TrimSpace(payload.Error))
	}
	if strings.TrimSpace(payload.Message.Content) == "" {
		return "", fmt.Errorf("Ollama 未返回摘要内容")
	}
	return strings.TrimSpace(payload.Message.Content), nil
}

type chatRequest struct {
	Model    string         `json:"model"`
	Stream   bool           `json:"stream"`
	Messages []chatMessage  `json:"messages"`
	Options  map[string]any `json:"options,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
	Error   string      `json:"error"`
}

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func joinURL(base, elem string) string {
	parsed, err := url.Parse(base)
	if err != nil {
		return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(elem, "/")
	}
	parsed.Path = path.Join(parsed.Path, elem)
	return parsed.String()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("复制本地文件失败: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("关闭目标文件失败: %w", err)
	}

	return nil
}

func downloadFile(dst, source string) error {
	resp, err := http.Get(source)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}

	tmpDst := dst + ".part"
	_ = os.Remove(tmpDst)
	f, err := os.Create(tmpDst)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(tmpDst)
		_ = f.Close()
		return fmt.Errorf("下载中断: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpDst)
		return fmt.Errorf("关闭文件失败: %w", err)
	}
	if err := os.Rename(tmpDst, dst); err != nil {
		_ = os.Remove(tmpDst)
		return fmt.Errorf("替换目标文件失败: %w", err)
	}
	return nil
}
