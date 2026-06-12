// =============================================================================
// MyMeetily — 全局应用状态管理 (useReducer + Context)
// =============================================================================

import { createContext, useContext, Dispatch } from 'react';
import type {
  AppPhase, DeviceInfo, DependencyStatus, MeetingState,
  StageStatus, TranscriptLine
} from '../types';

// ---- State Shape ----
export interface AppState {
  phase: AppPhase;

  // Devices
  micDevices: DeviceInfo[];
  speakerDevices: DeviceInfo[];
  selectedMic: string;
  selectedSpeaker: string;

  // Dependencies
  deps: DependencyStatus | null;
  appInfo: { engine: string; whisperModel: string; llmModel: string } | null;

  // Recording
  isRecording: boolean;
  elapsed: number;
  peakLevel: number;
  transcripts: TranscriptLine[];
  transcriptIndex: number;

  // Processing
  stages: StageStatus[];
  currentStage: string;

  // Result
  meetingState: MeetingState | null;
  actionStatus: string;
  actionError: boolean;
}

// ---- Default State ----
export const defaultStages: StageStatus[] = [
  { name: '音频预处理', status: 'pending' },
  { name: '语音识别',     status: 'pending' },
  { name: 'LLM 整理',    status: 'pending' },
  { name: '生成输出',     status: 'pending' },
];

export const initialState: AppState = {
  phase: 'checking',
  micDevices: [],
  speakerDevices: [],
  selectedMic: '',
  selectedSpeaker: '',
  deps: null,
  appInfo: null,
  isRecording: false,
  elapsed: 0,
  peakLevel: 0,
  transcripts: [],
  transcriptIndex: 0,
  stages: [...defaultStages],
  currentStage: '',
  meetingState: null,
  actionStatus: '',
  actionError: false,
};

// ---- Actions ----
export type AppAction =
  | { type: 'SET_PHASE'; phase: AppPhase }
  | { type: 'SET_MIC_DEVICES'; devices: DeviceInfo[] }
  | { type: 'SET_SPEAKER_DEVICES'; devices: DeviceInfo[] }
  | { type: 'SELECT_MIC'; device: string }
  | { type: 'SELECT_SPEAKER'; device: string }
  | { type: 'SET_DEPS'; deps: DependencyStatus; appInfo: AppState['appInfo'] }
  | { type: 'RECORDING_STARTED'; outputPath: string }
  | { type: 'RECORDING_STOPPED' }
  | { type: 'SET_PEAK_LEVEL'; level: number; elapsed: number }
  | { type: 'ADD_TRANSCRIPT'; text: string }
  | { type: 'SET_CURRENT_STAGE'; stage: string }
  | { type: 'PIPELINE_DONE'; meetingState: MeetingState }
  | { type: 'PIPELINE_ERROR'; error: string }
  | { type: 'SET_ACTION_STATUS'; status: string; error: boolean }
  | { type: 'RESET' };

// ---- Reducer ----
export function appReducer(state: AppState, action: AppAction): AppState {
  switch (action.type) {
    case 'SET_PHASE':
      return { ...state, phase: action.phase };

    case 'SET_MIC_DEVICES':
      return { ...state, micDevices: action.devices, selectedMic: action.devices[0]?.id || '' };

    case 'SET_SPEAKER_DEVICES':
      return { ...state, speakerDevices: action.devices };

    case 'SELECT_MIC':
      return { ...state, selectedMic: action.device };

    case 'SELECT_SPEAKER':
      return { ...state, selectedSpeaker: action.device };

    case 'SET_DEPS':
      return {
        ...state,
        deps: action.deps,
        appInfo: action.appInfo,
        phase: action.deps.summaryEnabled ? 'ready' : 'ready',
      };

    case 'RECORDING_STARTED':
      return {
        ...state,
        phase: 'recording',
        isRecording: true,
        elapsed: 0,
        peakLevel: 0,
        transcripts: [],
        transcriptIndex: 0,
      };

    case 'RECORDING_STOPPED':
      return { ...state, isRecording: false };

    case 'SET_PEAK_LEVEL':
      return { ...state, peakLevel: action.level, elapsed: action.elapsed };

    case 'ADD_TRANSCRIPT': {
      const nextIndex = state.transcriptIndex + 1;
      const newTranscripts = [...state.transcripts, { text: action.text, index: nextIndex }];
      // Keep last 100 lines
      const trimmed = newTranscripts.length > 100
        ? newTranscripts.slice(newTranscripts.length - 100)
        : newTranscripts;
      return { ...state, transcripts: trimmed, transcriptIndex: nextIndex };
    }

    case 'SET_CURRENT_STAGE': {
      const stageIndex = state.stages.findIndex(s => s.name === action.stage);
      const updatedStages = state.stages.map((s, i) => ({
        ...s,
        status: i < stageIndex ? 'done' as const : i === stageIndex ? 'active' as const : 'pending' as const,
      }));
      return { ...state, currentStage: action.stage, stages: updatedStages };
    }

    case 'PIPELINE_DONE':
      return {
        ...state,
        phase: 'result',
        meetingState: action.meetingState,
        stages: state.stages.map(s => ({ ...s, status: 'done' as const })),
      };

    case 'PIPELINE_ERROR':
      return {
        ...state,
        phase: 'ready',
        actionStatus: action.error,
        actionError: true,
        meetingState: null,
      };

    case 'SET_ACTION_STATUS':
      return { ...state, actionStatus: action.status, actionError: action.error };

    case 'RESET':
      return {
        ...initialState,
        phase: 'ready',
        micDevices: state.micDevices,
        speakerDevices: state.speakerDevices,
        selectedMic: state.selectedMic,
        selectedSpeaker: state.selectedSpeaker,
        deps: state.deps,
        appInfo: state.appInfo,
      };

    default:
      return state;
  }
}

// ---- Context ----
export const AppStateContext = createContext<AppState>(initialState);
export const AppDispatchContext = createContext<Dispatch<AppAction>>(() => {});

export function useAppState() {
  return useContext(AppStateContext);
}

export function useAppDispatch() {
  return useContext(AppDispatchContext);
}
