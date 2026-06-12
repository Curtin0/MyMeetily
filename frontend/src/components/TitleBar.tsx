// =============================================================================
// MyMeetily — TitleBar
// =============================================================================

import React from 'react';
import { useAppState } from '../hooks/useAppState';

const STYLE: Record<string, React.CSSProperties> = {
  bar: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '12px 24px',
    background: '#ffffff',
    borderBottom: '1px solid #e2e8f0',
    flexShrink: 0,
  },
  brand: {
    display: 'flex',
    alignItems: 'center',
    gap: 12,
  },
  logo: {
    width: 28,
    height: 28,
    borderRadius: 6,
    background: '#3b82f6',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: '#fff',
    fontSize: 14,
    fontWeight: 700,
  },
  title: {
    fontSize: 18,
    fontWeight: 700,
    color: '#1e293b',
    letterSpacing: '-0.5px',
  },
  subtitle: {
    fontSize: 12,
    color: '#64748b',
    marginLeft: 4,
  },
  status: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: '50%',
    display: 'inline-block',
  },
};

export const TitleBar: React.FC = () => {
  const state = useAppState();

  const getStatusColor = () => {
    switch (state.phase) {
      case 'checking': return '#f59e0b'; // amber
      case 'recording': return '#ef4444'; // red
      case 'processing': return '#3b82f6'; // blue
      case 'result': return '#10b981'; // green
      default: return '#94a3b8'; // gray
    }
  };

  const getStatusText = () => {
    switch (state.phase) {
      case 'checking': return '检查中';
      case 'ready': return '就绪';
      case 'recording': return '录制中';
      case 'processing': return '处理中';
      case 'result': return '完成';
    }
  };

  return (
    <div style={STYLE.bar}>
      <div style={STYLE.brand}>
        <div style={STYLE.logo}>M</div>
        <div>
          <span style={STYLE.title}>MyMeetily</span>
          <span style={STYLE.subtitle}>本地离线AI会议助手</span>
        </div>
      </div>
      <div style={STYLE.status}>
        <span style={{ ...STYLE.dot, background: getStatusColor() }} />
        <span style={{ fontSize: 13, color: '#475569' }}>{getStatusText()}</span>
        {state.deps && (
          <span style={{
            fontSize: 11,
            color: state.deps.summaryEnabled ? '#10b981' : '#94a3b8',
            padding: '2px 8px',
            background: state.deps.summaryEnabled ? '#f0fdf4' : '#f8fafc',
            borderRadius: 4,
            border: `1px solid ${state.deps.summaryEnabled ? '#bbf7d0' : '#e2e8f0'}`,
          }}>
            {state.deps.summaryEnabled ? 'AI摘要' : '仅转写'}
          </span>
        )}
      </div>
    </div>
  );
};
