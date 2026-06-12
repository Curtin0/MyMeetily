// =============================================================================
// MyMeetily — StatusPanel (依赖状态)
// =============================================================================

import React from 'react';
import { useAppState } from '../hooks/useAppState';

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
  item: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    padding: '4px 0',
    fontSize: 12,
    color: '#475569',
  },
  info: {
    fontSize: 11,
    color: '#94a3b8',
    marginTop: 8,
    lineHeight: 1.5,
  },
};

function dotStyle(ok: boolean): React.CSSProperties {
  return {
    width: 6,
    height: 6,
    borderRadius: '50%',
    background: ok ? '#10b981' : '#94a3b8',
    flexShrink: 0,
  };
}

export const StatusPanel: React.FC = () => {
  const { deps, appInfo } = useAppState();

  if (!deps) {
    return (
      <div style={STYLE.panel}>
        <div style={STYLE.title}>系统状态</div>
        <div style={{ fontSize: 12, color: '#94a3b8', textAlign: 'center', padding: 12 }}>
          正在检查...
        </div>
      </div>
    );
  }

  return (
    <div style={STYLE.panel}>
      <div style={STYLE.title}>系统状态</div>
      <div style={STYLE.item}>
        <span style={dotStyle(true)} />
        <span>whisper.cpp 引擎</span>
      </div>
      <div style={STYLE.item}>
        <span style={dotStyle(deps.whisperModel.ok)} />
        <span>语音模型: {deps.whisperModel.ok ? '已就绪' : '未找到'}</span>
      </div>
      <div style={STYLE.item}>
        <span style={dotStyle(deps.ollama.ok)} />
        <span>Ollama: {deps.ollama.ok ? '已连接' : '未连接'}</span>
      </div>
      {appInfo && (
        <div style={STYLE.info}>
          {appInfo.whisperModel} | {appInfo.llmModel}
        </div>
      )}
    </div>
  );
};
