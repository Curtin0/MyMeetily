package audio

import (
	"path/filepath"
	"time"
)

// Recorder captures audio from one or more sources and writes it to a file.
type Recorder interface {
	Start() error
	Stop() error
	IsRunning() bool
	Elapsed() time.Duration
	OutputPath() string
	// PeakLevel returns the current mic input level (0.0 = silent, 1.0 = clipping).
	PeakLevel() float32
}

// Backend identifies the recording implementation.
type Backend string

const BackendWASAPI Backend = "wasapi"

// AudioFormat describes the PCM format of captured audio.
type AudioFormat struct {
	SampleRate    int
	BitsPerSample int
	Channels      int
}

// RecorderConfig carries recorder settings.
type RecorderConfig struct {
	MicDevice     string
	SpeakerDevice string
	OutputPath    string
	// OnAudioData is an optional callback invoked with raw PCM data from the mic
	// and the actual capture format. Called from the capture goroutine; must be fast.
	OnAudioData func([]byte, AudioFormat)
}

// NewRecorder creates a WASAPI recorder.
func NewRecorder(cfg RecorderConfig) (Recorder, error) {
	return NewWASAPIRecorder(cfg)
}

// ListCaptureDevices returns selectable microphone devices.
func ListCaptureDevices(_ Backend) ([]string, error) {
	return listWASAPICaptureDevices()
}

// ListLoopbackDevices returns selectable system-audio capture devices.
func ListLoopbackDevices(_ Backend) ([]string, error) {
	return listWASAPILoopbackDevices()
}

// BuildRecordingPath creates the target recording path under a timestamp-based
// subdirectory, e.g. output/20260601_221540/live_20260601_221540.wav.
func BuildRecordingPath(outputDir string, recordedAt time.Time) string {
	ts := recordedAt.Format("20060102_150405")
	return filepath.Join(outputDir, ts, "live_"+ts+".wav")
}
