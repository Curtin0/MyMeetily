// =============================================================================
// MyMeetily — LiveView (录音实时界面)
// =============================================================================

import React from 'react';
import { useAppState } from '../hooks/useAppState';
import { AudioMeter } from './AudioMeter';
import { TranscriptionFeed } from './TranscriptionFeed';

const STYLE: Record<string, React.CSSProperties> = {
  container: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    background: '#ffffff',
    borderRadius: 8,
    border: '1px solid #e2e8f0',
    overflow: 'hidden',
  },
  header: {
    padding: '16px 20px',
    borderBottom: '1px solid #f1f5f9',
  },
  title: {
    fontSize: 15,
    fontWeight: 600,
    color: '#1e293b',
  },
  subtitle: {
    fontSize: 12,
    color: '#64748b',
    marginTop: 2,
  },
  body: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
  },
};

export const LiveView: React.FC = () => {
  const { isRecording } = useAppState();

  if (!isRecording) return null;

  return (
    <div style={STYLE.container}>
      <div style={STYLE.header}>
        <div style={STYLE.title}>实时录音</div>
        <div style={STYLE.subtitle}>AI 实时转写中...</div>
      </div>
      <div style={STYLE.body}>
        <AudioMeter />
        <TranscriptionFeed />
      </div>
    </div>
  );
};
