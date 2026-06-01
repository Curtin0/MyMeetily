//go:build !windows

package audio

import "fmt"

func NewWASAPIRecorder(cfg RecorderConfig) (Recorder, error) {
	return nil, fmt.Errorf("WASAPI recorder is only supported on Windows")
}