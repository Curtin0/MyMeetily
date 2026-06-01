//go:build windows

package audio

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

const wasapiLoopbackBufferDuration = wca.REFERENCE_TIME(400 * 10000)

type wasapiRecorder struct {
	mu            sync.Mutex
	micDevice     string
	speakerDevice string
	outputPath    string
	running       bool
	startedAt     time.Time
	cancel        context.CancelFunc
	sessions      []*wasapiCaptureSession
	tempDir       string
	peakLevel        float32    // mic peak
	speakerPeakLevel float32    // speaker loopback peak
	onAudioData      func([]byte, AudioFormat) // tee callback for live transcription
}

type wasapiCaptureSession struct {
	name        string
	selection   string
	flow        uint32
	loopback    bool
	filePath    string
	ready       chan error
	done        chan struct{}
	err         error
	micPeak     *float32   // pointer to recorder.peakLevel, nil for speaker sessions
	onAudioData func([]byte, AudioFormat) // tee callback for live transcription
}

func NewWASAPIRecorder(cfg RecorderConfig) (Recorder, error) {
	if strings.TrimSpace(cfg.MicDevice) == "" {
		return nil, fmt.Errorf("WASAPI 录音需要选择麦克风设备")
	}

	return &wasapiRecorder{
		micDevice:    cfg.MicDevice,
		speakerDevice: cfg.SpeakerDevice,
		outputPath:   cfg.OutputPath,
		onAudioData:  cfg.OnAudioData,
	}, nil
}

func (r *wasapiRecorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("already recording")
	}

	if err := ensureParentDir(r.outputPath); err != nil {
		return fmt.Errorf("prepare recording output: %w", err)
	}

	tempDir, err := os.MkdirTemp(filepath.Dir(r.outputPath), "wasapi-record-*")
	if err != nil {
		return fmt.Errorf("create temp capture dir: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sessions := []*wasapiCaptureSession{
		{
			name:        "mic",
			selection:   r.micDevice,
			flow:        wca.ECapture,
			filePath:    filepath.Join(tempDir, "mic.wav"),
			ready:       make(chan error, 1),
			done:        make(chan struct{}),
			micPeak:     &r.peakLevel,
			onAudioData: r.onAudioData,
		},
	}
	if strings.TrimSpace(r.speakerDevice) != "" {
		sessions = append(sessions, &wasapiCaptureSession{
			name:        "speaker",
				micPeak:     &r.speakerPeakLevel,
			selection: r.speakerDevice,
			flow:      wca.ERender,
			loopback:  true,
			filePath:  filepath.Join(tempDir, "speaker.wav"),
			ready:     make(chan error, 1),
			done:      make(chan struct{}),
		})
	}

	r.tempDir = tempDir
	r.cancel = cancel
	r.sessions = sessions

	for _, session := range sessions {
		go session.run(ctx)
	}

	for _, session := range sessions {
		if err := <-session.ready; err != nil {
			cancel()
			r.mu.Unlock()
			waitForWASAPISessions(sessions)
			r.mu.Lock()
			_ = os.RemoveAll(tempDir)
			r.tempDir = ""
			r.cancel = nil
			r.sessions = nil
			return fmt.Errorf("启动 %s WASAPI 录音失败: %w", session.name, err)
		}
	}

	r.running = true
	r.startedAt = time.Now()
	return nil
}

func (r *wasapiRecorder) Stop() error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}

	cancel := r.cancel
	sessions := append([]*wasapiCaptureSession(nil), r.sessions...)
	tempDir := r.tempDir
	outputPath := r.outputPath
	r.running = false
	r.cancel = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	waitForWASAPISessions(sessions)

	for _, session := range sessions {
		if session.err != nil {
			_ = os.RemoveAll(tempDir)
			return fmt.Errorf("%s WASAPI 录音失败: %w", session.name, session.err)
		}
	}

	if err := finalizeWASAPIOutput(sessions, outputPath); err != nil {
		_ = os.RemoveAll(tempDir)
		return err
	}

	_ = os.RemoveAll(tempDir)

	info, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("recorded audio file not created: %s", outputPath)
	}
	if info.Size() == 0 {
		return fmt.Errorf("recorded audio file is empty: %s", outputPath)
	}

	r.mu.Lock()
	r.sessions = nil
	r.tempDir = ""
	r.mu.Unlock()
	return nil
}

func (r *wasapiRecorder) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

func (r *wasapiRecorder) Elapsed() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return 0
	}
	return time.Since(r.startedAt)
}

func (r *wasapiRecorder) OutputPath() string {
	return r.outputPath
}

// PeakLevel returns the max of mic and speaker loopback levels.
func (r *wasapiRecorder) PeakLevel() float32 {
	if r.speakerPeakLevel > r.peakLevel {
		return r.speakerPeakLevel
	}
	return r.peakLevel
}

func (s *wasapiCaptureSession) run(ctx context.Context) {
	defer close(s.done)
	s.err = s.capture(ctx)
}

func (s *wasapiCaptureSession) capture(ctx context.Context) error {
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		s.ready <- err
		return err
	}
	defer ole.CoUninitialize()

	enumerator, err := newWASAPIDeviceEnumerator()
	if err != nil {
		s.ready <- err
		return err
	}
	defer enumerator.Release()

	device, descriptor, err := findWASAPIDeviceBySelection(enumerator, s.flow, s.selection)
	if err != nil {
		s.ready <- err
		return err
	}
	defer device.Release()

	client, format, latency, err := initializeWASAPIAudioClient(device, s.loopback)
	if err != nil {
		s.ready <- err
		return fmt.Errorf("初始化 %s: %w", descriptor.Label, err)
	}
	defer client.Release()
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(format)))

	writer, err := newWAVFileWriter(s.filePath, format.NSamplesPerSec, format.WBitsPerSample, format.NChannels)
	if err != nil {
		s.ready <- err
		return err
	}
	defer writer.Close()

	var captureClient *wca.IAudioCaptureClient
	if err := client.GetService(wca.IID_IAudioCaptureClient, &captureClient); err != nil {
		s.ready <- err
		return err
	}
	defer captureClient.Release()

	if err := client.Start(); err != nil {
		s.ready <- err
		return err
	}
	defer client.Stop()

	time.Sleep(latency)
	s.ready <- nil

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := captureWASAPIPacket(captureClient, writer, format, s.micPeak, s.onAudioData); err != nil {
			return err
		}

		if latency > 0 {
			time.Sleep(latency / 2)
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func waitForWASAPISessions(sessions []*wasapiCaptureSession) {
	for _, session := range sessions {
		<-session.done
	}
}

func initializeWASAPIAudioClient(device *wca.IMMDevice, loopback bool) (*wca.IAudioClient, *wca.WAVEFORMATEX, time.Duration, error) {
	var client *wca.IAudioClient
	if err := device.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &client); err != nil {
		return nil, nil, 0, err
	}

	var format *wca.WAVEFORMATEX
	if err := client.GetMixFormat(&format); err != nil {
		client.Release()
		return nil, nil, 0, err
	}

	format.WFormatTag = 1
	format.NBlockAlign = (format.WBitsPerSample / 8) * format.NChannels
	format.NAvgBytesPerSec = format.NSamplesPerSec * uint32(format.NBlockAlign)
	format.CbSize = 0

	var defaultPeriod wca.REFERENCE_TIME
	var minimumPeriod wca.REFERENCE_TIME
	if err := client.GetDevicePeriod(&defaultPeriod, &minimumPeriod); err != nil {
		ole.CoTaskMemFree(uintptr(unsafe.Pointer(format)))
		client.Release()
		return nil, nil, 0, err
	}

	latency := time.Duration(int(defaultPeriod) * 100)
	streamFlags := uint32(0)
	bufferDuration := defaultPeriod
	if loopback {
		streamFlags = wca.AUDCLNT_STREAMFLAGS_LOOPBACK
		bufferDuration = wasapiLoopbackBufferDuration
	}

	if err := client.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, streamFlags, bufferDuration, 0, format, nil); err != nil {
		ole.CoTaskMemFree(uintptr(unsafe.Pointer(format)))
		client.Release()
		return nil, nil, 0, err
	}

	return client, format, latency, nil
}

func captureWASAPIPacket(captureClient *wca.IAudioCaptureClient, writer *wavFileWriter, format *wca.WAVEFORMATEX, peakLevel *float32, onAudioData func([]byte, AudioFormat)) error {
	var data *byte
	var frames uint32
	var flags uint32
	var devicePosition uint64
	var qpcPosition uint64

	if err := captureClient.GetBuffer(&data, &frames, &flags, &devicePosition, &qpcPosition); err != nil {
		return nil
	}
	if frames == 0 {
		return nil
	}
	defer captureClient.ReleaseBuffer(frames)

	bytesPerFrame := int(format.NBlockAlign)
	byteCount := int(frames) * bytesPerFrame
	if byteCount <= 0 {
		return nil
	}

	buf := make([]byte, byteCount)
	if flags&wca.AUDCLNT_BUFFERFLAGS_SILENT != 0 || data == nil {
		if peakLevel != nil {
			*peakLevel = 0
		}
		return writer.Write(buf)
	}

	start := unsafe.Pointer(data)
	for index := 0; index < byteCount; index++ {
		buf[index] = *(*byte)(unsafe.Pointer(uintptr(start) + uintptr(index)))
	}

	// Compute peak level from captured samples.
	if peakLevel != nil {
		*peakLevel = computePeakLevel(buf, format)
	}

	// Tee audio data to live transcriber (non-blocking).
	if onAudioData != nil {
		onAudioData(buf, AudioFormat{
			SampleRate:    int(format.NSamplesPerSec),
			BitsPerSample: int(format.WBitsPerSample),
			Channels:      int(format.NChannels),
		})
	}

	return writer.Write(buf)
}

// computePeakLevel returns the maximum absolute sample value (0.0–1.0) from raw PCM data.
func computePeakLevel(buf []byte, format *wca.WAVEFORMATEX) float32 {
	bytesPerSample := int(format.WBitsPerSample / 8)
	if bytesPerSample == 0 || len(buf) < bytesPerSample {
		return 0
	}

	var maxAbs float32

	switch format.WBitsPerSample {
	case 16:
		for i := 0; i+2 <= len(buf); i += 2 {
			v := int16(binary.LittleEndian.Uint16(buf[i:]))
			abs := float32(v) / 32768.0
			if abs < 0 {
				abs = -abs
			}
			if abs > maxAbs {
				maxAbs = abs
			}
		}
	case 32:
		if format.WFormatTag == 3 { // IEEE float
			for i := 0; i+4 <= len(buf); i += 4 {
				v := math.Float32frombits(binary.LittleEndian.Uint32(buf[i:]))
				if v < 0 {
					v = -v
				}
				if v > maxAbs {
					maxAbs = v
				}
			}
		} else { // 32-bit integer PCM
			for i := 0; i+4 <= len(buf); i += 4 {
				v := int32(binary.LittleEndian.Uint32(buf[i:]))
				abs := float32(v) / 2147483648.0
				if abs < 0 {
					abs = -abs
				}
				if abs > maxAbs {
					maxAbs = abs
				}
			}
		}
	}

	if maxAbs > 1.0 {
		maxAbs = 1.0
	}
	return maxAbs
}

// finalizeWASAPIOutput produces the final output WAV from capture sessions.
// For a single session, the captured WAV is moved directly.
// For dual sessions (mic + speaker), PCM data is mixed in pure Go.
func finalizeWASAPIOutput(sessions []*wasapiCaptureSession, outputPath string) error {
	if len(sessions) == 0 {
		return fmt.Errorf("没有可用的 WASAPI 录音数据")
	}

	if len(sessions) == 1 {
		return os.Rename(sessions[0].filePath, outputPath)
	}

	var micPath, speakerPath string
	for _, session := range sessions {
		switch session.name {
		case "mic":
			micPath = session.filePath
		case "speaker":
			speakerPath = session.filePath
		}
	}

	return mixWAVToPath(micPath, speakerPath, outputPath)
}

// mixWAVToPath reads two WAV files, mixes their PCM samples, and writes the result.
func mixWAVToPath(pathA, pathB, outputPath string) error {
	samplesA, srA, chA, bitsA, err := readWAV(pathA)
	if err != nil {
		return fmt.Errorf("读取麦克风录音: %w", err)
	}
	samplesB, srB, chB, bitsB, err := readWAV(pathB)
	if err != nil {
		return fmt.Errorf("读取系统音频录音: %w", err)
	}

	sampleRate := srA
	channels := chA
	bitsPerSample := bitsA
	if srB != srA || chB != chA || bitsB != bitsA {
		return fmt.Errorf("音频格式不一致: mic=%dHz/%dch/%dbit speaker=%dHz/%dch/%dbit",
			srA, chA, bitsA, srB, chB, bitsB)
	}

	mixed := make([]int16, len(samplesA))
	if len(samplesB) > len(mixed) {
		mixed = make([]int16, len(samplesB))
		copy(mixed, samplesA)
	} else {
		copy(mixed, samplesA)
	}

	for i := 0; i < len(samplesB); i++ {
		v := int32(mixed[i]) + int32(samplesB[i])
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		mixed[i] = int16(v)
	}

	return writeWAVSamples(outputPath, mixed, sampleRate, channels, bitsPerSample)
}

// readWAV reads a WAV file and returns PCM samples as int16 values.
// Supports 16-bit PCM, 32-bit integer PCM, and 32-bit IEEE float.
func readWAV(path string) (samples []int16, sampleRate, channels, bitsPerSample int, err error) {
	var bitsPerSampleOrig int
	var audioFormatOrig uint16
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if len(data) < 44 {
		return nil, 0, 0, 0, fmt.Errorf("WAV 文件太小")
	}

	pos := 12 // skip "RIFF" header
	for pos < len(data)-8 {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))

		switch chunkID {
		case "fmt ":
			audioFormat := binary.LittleEndian.Uint16(data[pos+8 : pos+10])
			if audioFormat != 1 && audioFormat != 3 {
				return nil, 0, 0, 0, fmt.Errorf("只支持 PCM/IEEE float 格式，当前格式=%d", audioFormat)
			}
			channels = int(binary.LittleEndian.Uint16(data[pos+10 : pos+12]))
			sampleRate = int(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))
			bps := int(binary.LittleEndian.Uint16(data[pos+22 : pos+24]))
			if bps != 16 && bps != 32 {
				return nil, 0, 0, 0, fmt.Errorf("只支持 16-bit 或 32-bit，当前=%d", bps)
			}
			// Always report 16-bit since we normalize to int16 below.
			bitsPerSample = 16
			bitsPerSampleOrig = bps
			audioFormatOrig = audioFormat

		case "data":
			pcmStart := pos + 8
			pcmEnd := pcmStart + chunkSize
			if pcmEnd > len(data) {
				pcmEnd = len(data)
			}
			pcm := data[pcmStart:pcmEnd]

			switch {
			case bitsPerSampleOrig == 16:
				samples = make([]int16, len(pcm)/2)
				for i := range samples {
					samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
				}
			case bitsPerSampleOrig == 32 && audioFormatOrig == 1: // 32-bit integer PCM
				samples = make([]int16, len(pcm)/4)
				for i := range samples {
					v := int32(binary.LittleEndian.Uint32(pcm[i*4 : i*4+4]))
					samples[i] = int16(v >> 16)
				}
			case bitsPerSampleOrig == 32 && audioFormatOrig == 3: // 32-bit IEEE float
				samples = make([]int16, len(pcm)/4)
				for i := range samples {
					f := math.Float32frombits(binary.LittleEndian.Uint32(pcm[i*4 : i*4+4]))
					v := int32(f * 32767.0)
					if v > 32767 {
						v = 32767
					} else if v < -32768 {
						v = -32768
					}
					samples[i] = int16(v)
				}
			}
			return samples, sampleRate, channels, bitsPerSample, nil
		}

		pos += 8 + chunkSize
	}

	return nil, 0, 0, 0, fmt.Errorf("WAV 中未找到 data chunk")
}

// writeWAVSamples writes PCM int16 samples to a WAV file.
func writeWAVSamples(path string, samples []int16, sampleRate, channels, bitsPerSample int) error {
	writer, err := newWAVFileWriter(path, uint32(sampleRate), uint16(bitsPerSample), uint16(channels))
	if err != nil {
		return err
	}
	defer writer.Close()

	pcm := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(s))
	}

	return writer.Write(pcm)
}

type wavFileWriter struct {
	file          *os.File
	sampleRate    uint32
	bitsPerSample uint16
	channels      uint16
	dataSize      uint32
}

func newWAVFileWriter(path string, sampleRate uint32, bitsPerSample uint16, channels uint16) (*wavFileWriter, error) {
	if err := ensureParentDir(path); err != nil {
		return nil, err
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	writer := &wavFileWriter{
		file:          file,
		sampleRate:    sampleRate,
		bitsPerSample: bitsPerSample,
		channels:      channels,
	}

	if err := writer.writeHeader(); err != nil {
		file.Close()
		return nil, err
	}

	return writer, nil
}

func (w *wavFileWriter) Write(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	written, err := w.file.Write(data)
	if err != nil {
		return err
	}
	w.dataSize += uint32(written)
	return nil
}

func (w *wavFileWriter) Close() error {
	if w.file == nil {
		return nil
	}

	if _, err := w.file.Seek(0, 0); err != nil {
		w.file.Close()
		w.file = nil
		return err
	}
	if err := w.writeHeader(); err != nil {
		w.file.Close()
		w.file = nil
		return err
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *wavFileWriter) writeHeader() error {
	blockAlign := w.channels * (w.bitsPerSample / 8)
	byteRate := w.sampleRate * uint32(blockAlign)

	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], 36+w.dataSize)
	copy(header[8:12], []byte("WAVE"))
	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], w.channels)
	binary.LittleEndian.PutUint32(header[24:28], w.sampleRate)
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	binary.LittleEndian.PutUint16(header[34:36], w.bitsPerSample)
	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], w.dataSize)

	_, err := w.file.Write(header)
	return err
}
