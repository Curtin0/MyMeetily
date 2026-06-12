// =============================================================================
// MyMeetily — TypeScript 类型定义
// =============================================================================

// ---- Application Phase ----
export type AppPhase = 'checking' | 'ready' | 'recording' | 'processing' | 'result';

// ---- Device ----
export interface DeviceInfo {
  id: string;
  name: string;
  type: 'mic' | 'speaker';
}

// ---- Dependency Status ----
export interface CheckResult {
  ok: boolean;
  message: string;
}

export interface DependencyStatus {
  whisperBinary: CheckResult;
  whisperModel: CheckResult;
  ollama: CheckResult;
  summaryEnabled: boolean;
}

// ---- App Info ----
export interface AppInfo {
  engine: string;
  whisperModel: string;
  llmModel: string;
}

// ---- Transcript Segment ----
export interface Segment {
  start: number;
  end: number;
  text: string;
  confidence: number;
}

// ---- Meeting State (from Go backend) ----
export interface MeetingState {
  audioFilePath: string;
  language: string;
  rawTranscript: string;
  meetingNotes: string;
  summaryContent: string;
  summaryError: string;
  summaryEnabled: boolean;
  audioDuration: number;
  outputPath: string;
  htmlOutputPath: string;
  transcriptPath: string;
  markdownPath: string;
  segments: Segment[];
  error: string;
}

// ---- Pipeline Progress ----
export interface StageStatus {
  name: string;
  status: 'pending' | 'active' | 'done';
}

// ---- Recording ----
export interface TranscriptLine {
  text: string;
  index: number;
}

// ---- Wails Event Payloads ----
export interface RecordStartedPayload {
  outputPath: string;
}

export interface RecordStoppedPayload {
  outputPath: string;
  duration: number;
}

export interface PeakLevelPayload {
  level: number;
  elapsed: number;
}

export interface TranscriptPayload {
  text: string;
}

export interface PipelineProgressPayload {
  stage: string;
}

export interface AppStatusPayload {
  phase: string;
  message: string;
  summaryEnabled?: boolean;
}
