// =============================================================================
// MyMeetily — TranscriptionFeed (实时转写滚动)
// =============================================================================

import React, { useEffect, useRef } from 'react';
import { useAppState } from '../hooks/useAppState';

const STYLE: Record<string, React.CSSProperties> = {
  container: {
    flex: 1,
    overflow: 'auto' as const,
    padding: '16px 20px',
  },
  line: {
    padding: '6px 0',
    fontSize: 14,
    color: '#334155',
    lineHeight: 1.6,
    borderBottom: '1px solid #f8fafc',
  },
  empty: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100%',
  },
  emptyText: {
    fontSize: 14,
    color: '#cbd5e1',
    textAlign: 'center' as const,
  },
  pulse: {
    display: 'inline-block',
    width: 6,
    height: 6,
    borderRadius: '50%',
    background: '#3b82f6',
    marginRight: 8,
    animation: 'pulse 1.5s ease-in-out infinite',
  },
};

export const TranscriptionFeed: React.FC = () => {
  const { transcripts } = useAppState();
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [transcripts.length]);

  if (transcripts.length === 0) {
    return (
      <div style={STYLE.container}>
        <div style={STYLE.empty}>
          <div style={STYLE.emptyText}>
            <span style={STYLE.pulse} />
            等待语音输入...
          </div>
        </div>
      </div>
    );
  }

  return (
    <div style={STYLE.container}>
      {transcripts.map((line, i) => (
        <div key={i} style={STYLE.line}>
          {line.text}
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  );
};
