// =============================================================================
// MyMeetily — ControlPanel (录音/停止按钮)
// =============================================================================

import React, { useState } from 'react';
import { useAppState, useAppDispatch } from '../hooks/useAppState';
import { RecordService, PipelineService, AppService } from '../hooks/useWailsEvents';

const STYLE: Record<string, React.CSSProperties> = {
  panel: {
    background: '#ffffff',
    borderRadius: 8,
    border: '1px solid #e2e8f0',
    padding: 16,
    marginBottom: 12,
  },
  title: {
    fontSize: 12,
    fontWeight: 600,
    color: '#64748b',
    textTransform: 'uppercase' as const,
    letterSpacing: '1px',
    marginBottom: 12,
  },
  timer: {
    fontSize: 32,
    fontWeight: 700,
    color: '#1e293b',
    textAlign: 'center' as const,
    fontVariantNumeric: 'tabular-nums',
    marginBottom: 16,
    fontFamily: "'SF Mono', 'Fira Code', monospace",
  },
  recordBtn: {
    width: '100%',
    padding: '12px 0',
    borderRadius: 8,
    border: 'none',
    background: '#3b82f6',
    color: '#fff',
    fontSize: 15,
    fontWeight: 600,
    cursor: 'pointer',
    marginBottom: 8,
  },
  stopBtn: {
    width: '100%',
    padding: '12px 0',
    borderRadius: 8,
    border: '1px solid #e2e8f0',
    background: '#fff',
    color: '#ef4444',
    fontSize: 15,
    fontWeight: 600,
    cursor: 'pointer',
    marginBottom: 8,
  },
  importBtn: {
    width: '100%',
    padding: '8px 0',
    borderRadius: 6,
    border: '1px solid #e2e8f0',
    background: '#f8fafc',
    color: '#64748b',
    fontSize: 12,
    cursor: 'pointer',
  },
  disabled: {
    opacity: 0.5,
    cursor: 'not-allowed' as const,
  },
  error: {
    fontSize: 12,
    color: '#ef4444',
    textAlign: 'center' as const,
    marginTop: 8,
  },
};

function formatTime(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) {
    return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  }
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

export const ControlPanel: React.FC = () => {
  const { phase, isRecording, elapsed, selectedMic } = useAppState();
  const dispatch = useAppDispatch();
  const [error, setError] = useState('');
  const [stopping, setStopping] = useState(false);
  const [importing, setImporting] = useState(false);

  const handleStart = async () => {
    if (!window.go || !selectedMic) return;
    setError('');
    try {
      const speaker = ''; // Use selected speaker from state
      await RecordService().StartRecording(selectedMic, speaker);
    } catch (e: any) {
      setError(e?.message || '录音启动失败');
    }
  };

  const handleStop = async () => {
    if (!window.go) return;
    setStopping(true);
    setError('');
    try {
      const outputPath = await RecordService().StopRecording();
      if (outputPath) {
        // Start pipeline automatically
        dispatch({ type: 'SET_PHASE', phase: 'processing' });
        dispatch({ type: 'SET_CURRENT_STAGE', stage: '音频预处理' });
        try {
          await PipelineService().RunPipeline(outputPath);
        } catch (e: any) {
          // Pipeline error is handled by events
          setError(e?.message || '处理失败');
        }
      }
    } catch (e: any) {
      setError(e?.message || '停止录音失败');
    } finally {
      setStopping(false);
    }
  };

  const handleImport = async () => {
    if (!window.go) return;
    setImporting(true);
    setError('');
    try {
      const filePath = await AppService().OpenFileDialog();
      if (!filePath) {
        setImporting(false);
        return; // User cancelled
      }
      // Start pipeline with imported file
      dispatch({ type: 'SET_PHASE', phase: 'processing' });
      dispatch({ type: 'SET_CURRENT_STAGE', stage: '音频预处理' });
      try {
        await PipelineService().ImportAudioFile(filePath);
      } catch (e: any) {
        setError(e?.message || '处理失败');
      }
    } catch (e: any) {
      setError(e?.message || '导入失败');
    } finally {
      setImporting(false);
    }
  };

  return (
    <div style={STYLE.panel}>
      <div style={STYLE.title}>控制</div>

      {isRecording && (
        <div style={STYLE.timer}>{formatTime(elapsed)}</div>
      )}

      {!isRecording ? (
        <button
          style={{
            ...STYLE.recordBtn,
            ...(phase !== 'ready' || !selectedMic ? STYLE.disabled : {}),
          }}
          disabled={phase !== 'ready' || !selectedMic}
          onClick={handleStart}
        >
          ● 开始录音
        </button>
      ) : (
        <button
          style={STYLE.stopBtn}
          onClick={handleStop}
          disabled={stopping}
        >
          {stopping ? '正在保存...' : '■ 停止录音'}
        </button>
      )}

      {/* Import audio file — available when not recording */}
      {!isRecording && (
        <button
          style={{
            ...STYLE.importBtn,
            ...(importing || phase === 'processing' ? STYLE.disabled : {}),
          }}
          disabled={importing || phase === 'processing'}
          onClick={handleImport}
        >
          {importing ? '导入中...' : '📂 导入音频文件'}
        </button>
      )}

      {error && <div style={STYLE.error}>{error}</div>}
    </div>
  );
};
