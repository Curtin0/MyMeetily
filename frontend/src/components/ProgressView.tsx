// =============================================================================
// MyMeetily — ProgressView (管线处理进度)
// =============================================================================

import React from 'react';
import { useAppState } from '../hooks/useAppState';

const STYLE: Record<string, React.CSSProperties> = {
  container: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    background: '#ffffff',
    borderRadius: 8,
    border: '1px solid #e2e8f0',
    padding: 40,
  },
  title: {
    fontSize: 18,
    fontWeight: 600,
    color: '#1e293b',
    marginBottom: 24,
  },
  stages: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    marginBottom: 16,
    flexWrap: 'wrap' as const,
    justifyContent: 'center',
  },
  arrow: {
    color: '#cbd5e1',
    fontSize: 16,
  },
  spinner: {
    width: 24,
    height: 24,
    borderRadius: '50%',
    border: '3px solid #e2e8f0',
    borderTopColor: '#3b82f6',
    marginBottom: 12,
  },
  hint: {
    fontSize: 12,
    color: '#94a3b8',
  },
};

const ICONS: Record<string, string> = {
  done: '✓',
  active: '●',
  pending: '○',
};

function stageStyle(status: string): React.CSSProperties {
  return {
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    padding: '6px 12px',
    borderRadius: 6,
    fontSize: 13,
    fontWeight: status === 'active' ? 600 : 400,
    color: status === 'done' ? '#10b981' : status === 'active' ? '#3b82f6' : '#94a3b8',
    background: status === 'active' ? '#eff6ff' : status === 'done' ? '#f0fdf4' : '#f8fafc',
    border: `1px solid ${status === 'active' ? '#bfdbfe' : status === 'done' ? '#bbf7d0' : '#e2e8f0'}`,
  };
}

export const ProgressView: React.FC = () => {
  const { stages } = useAppState();

  return (
    <div style={STYLE.container}>
      <div style={STYLE.spinner} />
      <div style={STYLE.title}>AI 正在处理会议记录...</div>
      <div style={STYLE.stages}>
        {stages.map((stage, i) => (
          <React.Fragment key={stage.name}>
            <div style={stageStyle(stage.status)}>
              <span>{ICONS[stage.status]}</span>
              <span>{stage.name}</span>
            </div>
            {i < stages.length - 1 && (
              <span style={STYLE.arrow}>→</span>
            )}
          </React.Fragment>
        ))}
      </div>
      <div style={STYLE.hint}>处理过程中请勿关闭窗口</div>
    </div>
  );
};
