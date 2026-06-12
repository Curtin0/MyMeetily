// =============================================================================
// MyMeetily — AudioMeter (实时音量条)
// =============================================================================

import React from 'react';
import { useAppState } from '../hooks/useAppState';

const STYLE: Record<string, React.CSSProperties> = {
  container: {
    padding: '12px 20px',
    borderBottom: '1px solid #f1f5f9',
  },
  bar: {
    height: 8,
    background: '#f1f5f9',
    borderRadius: 4,
    overflow: 'hidden',
    marginBottom: 4,
  },
  info: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: 11,
    color: '#94a3b8',
  },
};

export const AudioMeter: React.FC = () => {
  const { peakLevel } = useAppState();

  const dbDisplay = peakLevel > 0 ? Math.round(20 * Math.log10(peakLevel)) : -Infinity;

  const fillStyle: React.CSSProperties = {
    height: '100%',
    width: `${Math.min(peakLevel * 100, 100)}%`,
    background: peakLevel > 0.8 ? '#ef4444' : peakLevel > 0.5 ? '#f59e0b' : '#3b82f6',
    borderRadius: 4,
    transition: 'width 60ms ease-out',
  };

  return (
    <div style={STYLE.container}>
      <div style={STYLE.bar}>
        <div style={fillStyle} />
      </div>
      <div style={STYLE.info}>
        <span>{dbDisplay === -Infinity ? '-∞ dB' : `${dbDisplay} dB`}</span>
        <span>{Math.round(peakLevel * 100)}%</span>
      </div>
    </div>
  );
};
