package agent

type Segment struct {
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

type MeetingState struct {
	AudioFilePath string `json:"audio_file_path"`
	Language      string `json:"language"`

	// ProgressFn is an optional callback for reporting pipeline stage progress.
	// It is set by the TUI layer and should not be serialized.
	ProgressFn func(stage string) `json:"-"`

	WavFilePath   string    `json:"wav_file_path"`
	RawTranscript string    `json:"raw_transcript"`
	Segments      []Segment `json:"segments"`
	AudioDuration float64   `json:"audio_duration"`

	MeetingNotes   string `json:"meeting_notes"`
	SummaryContent string `json:"summary_content,omitempty"`
	SummaryError   string `json:"summary_error,omitempty"`
	OutputPath     string `json:"output_path"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	HTMLOutputPath string `json:"html_output_path,omitempty"`
	MarkdownPath   string `json:"markdown_path,omitempty"`
	SummaryEnabled bool   `json:"summary_enabled"`
	Error          string `json:"error,omitempty"`
}
