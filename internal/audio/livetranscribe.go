package audio

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"bytes"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// LiveTranscriber performs real-time VAD-based sentence segmentation and
// asynchronous whisper.cpp transcription during recording.
type LiveTranscriber struct {
	cfg LiveTranscribeConfig

	audioCh chan []byte   // raw PCM from mic
	closeCh chan struct{} // signals that recording has stopped (flush pending)

	// Results: transcribed sentences are sent here for the TUI to consume.
	Results <-chan string
	lastResult string // dedup: skip consecutive identical transcripts
	results chan string

	// Diagnostics (updated atomically from VAD goroutine, read by UI).
	Diag LiveTranscribeDiag

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// LiveTranscribeDiag holds real-time diagnostic counters for the UI.
type LiveTranscribeDiag struct {
	AudioPackets   int64
	VoiceFrames    int64
	SilentFrames   int64
	SentencesCut   int64
	WhisperOK      int64
	WhisperEmpty   int64
	WhisperErr     int64
	LastResult     string
	WhisperRunning int64
	LastRMS        float32
}

// Snapshot returns a copy of the diagnostic counters.
func (d *LiveTranscribeDiag) Snapshot() LiveTranscribeDiag {
	return *d
}

func (d *LiveTranscribeDiag) incAudio()   { d.AudioPackets++ }
func (d *LiveTranscribeDiag) incVoice()   { d.VoiceFrames++ }
func (d *LiveTranscribeDiag) incSilent()  { d.SilentFrames++ }
func (d *LiveTranscribeDiag) incSentence(){ d.SentencesCut++ }
func (d *LiveTranscribeDiag) setRMS(v float32) { d.LastRMS = v }
func (d *LiveTranscribeDiag) addWhisper(n int64) { d.WhisperRunning += n }

// LiveTranscribeConfig carries the settings for real-time transcription.
type LiveTranscribeConfig struct {
	WhisperBinary string
	ModelPath     string
	Language      string
	SampleRate    int
	BitsPerSample int
	Channels      int
}

// NewLiveTranscriber creates a LiveTranscriber and starts its processing goroutines.
// The caller must call Close() when recording ends.
func NewLiveTranscriber(cfg LiveTranscribeConfig) *LiveTranscriber {
	ctx, cancel := context.WithCancel(context.Background())

	results := make(chan string, 32) // buffered so whisper goroutines never block the UI

	lt := &LiveTranscriber{
		cfg:     cfg,
		audioCh: make(chan []byte, 200), // ~4s buffer at 20ms/packet
		closeCh: make(chan struct{}),
		results: results,
		Results: results,
		cancel:  cancel,
	}

	lt.wg.Add(1)
	go lt.vadLoop(ctx)

	return lt
}

// Feed sends a chunk of raw PCM data to the transcriber. Called from the WASAPI
// capture goroutine. Must be non-blocking to avoid holding up the capture loop.
func (lt *LiveTranscriber) Feed(pcm []byte, af AudioFormat) {
	// Lazy-init format from first audio packet.
	if lt.cfg.SampleRate == 0 && af.SampleRate > 0 {
		lt.cfg.SampleRate = af.SampleRate
		lt.cfg.BitsPerSample = af.BitsPerSample
		lt.cfg.Channels = af.Channels
	}
	select {
	case lt.audioCh <- pcm:
	default:
	}
}

// Close signals the transcriber to flush any remaining audio and stop.
// It waits for all in-flight whisper processes to finish, then closes Results.
var debugLogMu sync.Mutex

func (lt *LiveTranscriber) writeDebugLog(msg string) {
	debugLogMu.Lock()
	defer debugLogMu.Unlock()
	f, err := os.OpenFile("transcriber_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil { return }
	defer f.Close()
	fmt.Fprintln(f, msg)
}
func (lt *LiveTranscriber) Close() {
	close(lt.closeCh)  // signal VAD to flush
	lt.cancel()        // signal workers to stop
	lt.wg.Wait()       // wait for goroutines
	close(lt.results)  // signal TUI that no more results will come
}

// --- VAD constants ---

const (
	vadWindowMs  = 30    // analysis window in ms
	vadThreshold = 0.015 // RMS energy threshold — low enough for consumer mics
)

var (
	vadSilenceMs  = 1000.0 // silence duration to mark sentence boundary
	vadMinUtterMs = 800.0 // minimum utterance duration to process
)

// vadLoop reads raw PCM from audioCh, runs energy-based VAD, cuts sentences,
// and enqueues WAV files for transcription.
func (lt *LiveTranscriber) vadLoop(ctx context.Context) {
	defer lt.wg.Done()

	// Wait for the first audio packet to set format.
	for lt.cfg.SampleRate == 0 {
		select {
		case <-ctx.Done():
			return
		case <-lt.closeCh:
			return
		case <-lt.audioCh:
			// Format was set by Feed; data is discarded since we don't have
			// format yet at this point, but it's just a tiny amount.
		}
	}

	bytesPerSample := lt.cfg.BitsPerSample / 8
	samplesPerFrame := int(float64(lt.cfg.SampleRate) * float64(vadWindowMs) / 1000.0)
	if samplesPerFrame < 1 {
		samplesPerFrame = 256
	}
	_ = samplesPerFrame * bytesPerSample * lt.cfg.Channels

	var (
		ringBuf     []byte            // accumulated audio for current utterance
		silentFrames int              // consecutive silent frames
		voiceActive bool
	)

	// VAD thresholds converted to sample/frame counts.
	minSamples := int(vadMinUtterMs / 1000.0 * float64(lt.cfg.SampleRate))
	maxSilentFrames := int(vadSilenceMs / float64(vadWindowMs))

	flushUtterance := func() {
		needed := minSamples * bytesPerSample * lt.cfg.Channels
		if len(ringBuf) < needed {
			ringBuf = nil
			return
		}
		sentenceWAV, err := writeSentenceWAV(ringBuf, lt.cfg)
		ringBuf = nil
		if err != nil {
			slog.Error("livetranscriber: write sentence WAV", "error", err)
			return
		}
		// Enqueue for whisper processing.
		select {
		case lt.results <- "__pending__":
		default:
		}
			lt.Diag.incSentence()
			lt.Diag.addWhisper(1)
		go lt.transcribeOne(ctx, sentenceWAV)
	}

	for {
		select {
		case <-ctx.Done():
			// Context cancelled: drain remaining and exit.
			if voiceActive && len(ringBuf) > 0 {
				flushUtterance()
			}
			return

		case <-lt.closeCh:
			// Recording stopped: flush and exit.
			if len(ringBuf) > 0 {
				flushUtterance()
			}
			return

		case pcm := <-lt.audioCh:
				lt.Diag.incAudio()
			if len(pcm) == 0 {
				continue
			}

			rms := computeRMS(pcm, lt.cfg.BitsPerSample)
			lt.Diag.setRMS(rms)
			isVoice := rms >= vadThreshold

			if isVoice {
			lt.Diag.incVoice()
				silentFrames = 0
				if !voiceActive {
					voiceActive = true
					ringBuf = nil // start fresh utterance
				}
				ringBuf = append(ringBuf, pcm...)
			lt.Diag.incSilent()
			} else if voiceActive {
				silentFrames++
				ringBuf = append(ringBuf, pcm...)
				if silentFrames >= maxSilentFrames {
					flushUtterance()
					voiceActive = false
					silentFrames = 0
				}
			}
			// If not voice active and not speaking, discard audio.
		}
	}
}

// transcribeOne runs whisper-cli on a single sentence WAV and sends the
// recognized text to lt.results.
func (lt *LiveTranscriber) transcribeOne(ctx context.Context, wavPath string) {
	defer os.Remove(wavPath)

	args := []string{
		"-m", lt.cfg.ModelPath,
		"-f", wavPath,
		"-l", lt.cfg.Language,
		"-oj",
	}

	info, _ := os.Stat(wavPath)
	wavSize := int64(0)
	if info != nil { wavSize = info.Size() }

	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, lt.cfg.WhisperBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	raw := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())

	lt.writeDebugLog(fmt.Sprintf("WHISPER wav=%s size=%d model=%s lang=%s err=%v stdout_len=%d stderr=%s", wavPath, wavSize, lt.cfg.ModelPath, lt.cfg.Language, runErr, len(raw), errStr))

	if runErr != nil {
		lt.Diag.addWhisper(-1)
		lt.Diag.WhisperErr++
		lt.writeDebugLog(fmt.Sprintf("WHISPER FAIL stderr=%s", errStr))
		return
	}
	text := extractWhisperText(raw)
	if text == "" {
		lt.Diag.WhisperEmpty++
		lt.writeDebugLog(fmt.Sprintf("WHISPER EMPTY json=%s", raw))
		return
	}
	lt.Diag.WhisperOK++
	if text == lt.lastResult { lt.Diag.WhisperEmpty++; return }
	lt.Diag.LastResult = text
	lt.lastResult = text
	lt.writeDebugLog(fmt.Sprintf("WHISPER OK text=%s", text))

	lt.Diag.addWhisper(-1)
	select {
	case lt.results <- text:
	case <-ctx.Done():
	}
}

// --- VAD helpers ---

// computeRMS calculates the RMS energy of a raw PCM buffer, normalized to 0-1.
func computeRMS(pcm []byte, bitsPerSample int) float32 {
	bytesPerSample := bitsPerSample / 8
	if bytesPerSample == 0 || len(pcm) < bytesPerSample {
		return 0
	}

	var sum float64
	count := len(pcm) / bytesPerSample

	for i := 0; i+bytesPerSample <= len(pcm); i += bytesPerSample {
		var val float64
		switch bitsPerSample {
		case 16:
			v := int16(binary.LittleEndian.Uint16(pcm[i:]))
			val = float64(v) / 32768.0
		case 32:
			// WASAPI shared mode gives 32-bit integer PCM (WFormatTag=1).
			v := int32(binary.LittleEndian.Uint32(pcm[i:]))
			val = float64(v) / 2147483648.0
		default:
			return 0
		}
		sum += val * val
	}

	if count == 0 {
		return 0
	}
	return float32(math.Sqrt(sum / float64(count)))
}

// writeSentenceWAV writes accumulated PCM bytes to a temporary WAV file
// suitable for whisper.cpp.
func writeSentenceWAV(pcm []byte, cfg LiveTranscribeConfig) (string, error) {
	f, err := os.CreateTemp("", "vad-sentence-*.wav")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	// Convert to 16-bit PCM if needed (whisper.cpp needs 16-bit or float).
	var pcm16 []byte
	if cfg.BitsPerSample == 32 {
		bytesPerSample := 4
		sampleCount := len(pcm) / bytesPerSample
		pcm16 = make([]byte, sampleCount*2)
		for i := 0; i < sampleCount; i++ {
			v := int32(binary.LittleEndian.Uint32(pcm[i*4:]))
			binary.LittleEndian.PutUint16(pcm16[i*2:], uint16(v>>16))
		}
	} else {
		pcm16 = pcm
	}

	outBits := 16
	totalDataLen := len(pcm16)
	byteRate := cfg.SampleRate * cfg.Channels * (outBits / 8)
	blockAlign := cfg.Channels * (outBits / 8)

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+totalDataLen))
	copy(header[8:16], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], uint16(cfg.Channels))
	binary.LittleEndian.PutUint32(header[24:], uint32(cfg.SampleRate))
	binary.LittleEndian.PutUint32(header[28:], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:], uint16(outBits))
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(totalDataLen))

	if _, err := f.Write(header); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if _, err := f.Write(pcm16); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}

	path := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

// extractWhisperText pulls the "text" field from whisper.cpp JSON output.
func extractWhisperText(raw string) string {
	// Try JSON format first: {"text":"..."}
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err == nil && result.Text != "" {
		return strings.TrimSpace(result.Text)
	}
	// Try timestamp format: [HH:MM:SS.mmm --> HH:MM:SS.mmm]  text
	if idx := strings.Index(raw, "]  "); idx >= 0 {
		return strings.TrimSpace(raw[idx+3:])
	}
	if idx := strings.Index(raw, "] "); idx >= 0 {
		return strings.TrimSpace(raw[idx+2:])
	}
	// Fallback: return raw text (may contain timestamps)
	return raw
}

func stripFence(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
