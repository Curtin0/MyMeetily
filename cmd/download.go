package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func findAppRoot() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		if fileExists(filepath.Join(currentDir, "go.mod")) {
			return currentDir
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "."
		}
		currentDir = parentDir
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// copyOrDownloadFile copies a local file or downloads a remote URL to dst.
func copyOrDownloadFile(dst, source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("source 不能为空")
	}

	if pathExists(source) {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}

		in, err := os.Open(source)
		if err != nil {
			return fmt.Errorf("打开本地文件失败: %w", err)
		}
		defer in.Close()

		out, err := os.Create(dst)
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			return fmt.Errorf("复制本地文件失败: %w", err)
		}
		return nil
	}

	return downloadFile(dst, source)
}

func downloadFile(dst, url string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := downloadFileOnce(dst, url); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < maxAttempts {
				fmt.Printf("  下载失败，正在重试 (%d/%d)...\n", attempt, maxAttempts)
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
			}
		}
	}

	return lastErr
}

func downloadFileOnce(dst, url string) error {
	tmpDst := dst + ".part"
	_ = os.Remove(tmpDst)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}

	total := resp.ContentLength

	f, err := os.Create(tmpDst)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	buf := make([]byte, 32*1024)
	var downloaded int64
	start := time.Now()
	lastPrint := time.Now()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				_ = os.Remove(tmpDst)
				return fmt.Errorf("写入文件失败: %w", werr)
			}
			downloaded += int64(n)

			if time.Since(lastPrint) > 500*time.Millisecond {
				elapsed := time.Since(start)
				speed := float64(downloaded) / elapsed.Seconds() / 1024 / 1024
				if total > 0 {
					pct := float64(downloaded) / float64(total) * 100
					fmt.Printf("\r  下载中... %.1f%%  (%.1f MB/s)     ", pct, speed)
				} else {
					mb := float64(downloaded) / 1024 / 1024
					fmt.Printf("\r  下载中... %.1f MB  (%.1f MB/s)     ", mb, speed)
				}
				lastPrint = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = os.Remove(tmpDst)
			return fmt.Errorf("下载中断: %w", err)
		}
	}

	if total > 0 {
		fmt.Printf("\r  下载完成 100.0%%                      \n")
	} else {
		fmt.Printf("\r  下载完成                              \n")
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

func pathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
