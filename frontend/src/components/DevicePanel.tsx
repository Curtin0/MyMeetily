// =============================================================================
// MyMeetily — DevicePanel (麦克风/扬声器选择)
// =============================================================================

import React, { useEffect, useState } from 'react';
import { useAppState, useAppDispatch } from '../hooks/useAppState';
import { DeviceService } from '../hooks/useWailsEvents';

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
  field: {
    marginBottom: 12,
  },
  label: {
    fontSize: 13,
    color: '#475569',
    marginBottom: 4,
    fontWeight: 500,
  },
  select: {
    width: '100%',
    padding: '8px 10px',
    borderRadius: 6,
    border: '1px solid #e2e8f0',
    background: '#f8fafc',
    fontSize: 13,
    color: '#334155',
    outline: 'none',
    boxSizing: 'border-box' as const,
  },
  button: {
    width: '100%',
    padding: '8px 0',
    borderRadius: 6,
    border: '1px solid #e2e8f0',
    background: '#f8fafc',
    fontSize: 12,
    color: '#64748b',
    cursor: 'pointer',
    marginBottom: 8,
  },
  hint: {
    fontSize: 11,
    color: '#94a3b8',
    marginTop: 4,
  },
  loading: {
    fontSize: 12,
    color: '#94a3b8',
    textAlign: 'center' as const,
    padding: 12,
  },
};

export const DevicePanel: React.FC = () => {
  const { micDevices, speakerDevices, selectedMic, selectedSpeaker, phase } = useAppState();
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);

  const loadDevices = async () => {
    setLoading(true);
    try {
      if (window.go) {
        const mics = await DeviceService().ListMicDevices();
        dispatch({ type: 'SET_MIC_DEVICES', devices: mics || [] });
        const speakers = await DeviceService().ListSpeakerDevices();
        dispatch({ type: 'SET_SPEAKER_DEVICES', devices: speakers || [] });
      }
    } catch (e) {
      console.error('Failed to load devices:', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (window.go && phase === 'ready') {
      loadDevices();
    }
  }, [phase]);

  const recording = phase === 'recording';

  return (
    <div style={STYLE.panel}>
      <div style={STYLE.title}>音频设备</div>

      {loading ? (
        <div style={STYLE.loading}>正在检测设备...</div>
      ) : (
        <>
          <div style={STYLE.field}>
            <div style={STYLE.label}>麦克风</div>
            <select
              style={{ ...STYLE.select, opacity: recording ? 0.6 : 1 }}
              value={selectedMic}
              onChange={e => dispatch({ type: 'SELECT_MIC', device: e.target.value })}
              disabled={recording}
            >
              {micDevices.length === 0 && (
                <option value="">未检测到设备</option>
              )}
              {micDevices.map(d => (
                <option key={d.id} value={d.id}>{d.name}</option>
              ))}
            </select>
          </div>

          <div style={STYLE.field}>
            <div style={STYLE.label}>系统音频 (扬声器回采)</div>
            <select
              style={{ ...STYLE.select, opacity: recording ? 0.6 : 1 }}
              value={selectedSpeaker}
              onChange={e => dispatch({ type: 'SELECT_SPEAKER', device: e.target.value })}
              disabled={recording}
            >
              <option value="">不录制系统声音</option>
              {speakerDevices.map(d => (
                <option key={d.id} value={d.id}>{d.name}</option>
              ))}
            </select>
          </div>

          <button
            style={STYLE.button}
            onClick={loadDevices}
            disabled={recording}
          >
            ↻ 刷新设备列表
          </button>

          <div style={STYLE.hint}>使用 WASAPI 采集音频</div>
        </>
      )}
    </div>
  );
};
