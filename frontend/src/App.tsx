// =============================================================================
// MyMeetily — Main Application Component
// =============================================================================

import React, { useReducer } from 'react';
import { appReducer, initialState, AppStateContext, AppDispatchContext } from './hooks/useAppState';
import { useWailsEvents } from './hooks/useWailsEvents';
import { TitleBar } from './components/TitleBar';
import { DevicePanel } from './components/DevicePanel';
import { ControlPanel } from './components/ControlPanel';
import { StatusPanel } from './components/StatusPanel';
import { LiveView } from './components/LiveView';
import { ProgressView } from './components/ProgressView';
import { ResultView } from './components/ResultView';

const STYLE: Record<string, React.CSSProperties> = {
  app: {
    display: 'flex',
    flexDirection: 'column',
    height: '100vh',
    background: '#f8fafc',
    color: '#334155',
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
  },
  main: {
    flex: 1,
    display: 'flex',
    overflow: 'hidden',
  },
  sidebar: {
    width: 260,
    padding: '16px',
    flexShrink: 0,
    overflow: 'auto',
    borderRight: '1px solid #e2e8f0',
    background: '#f8fafc',
  },
  content: {
    flex: 1,
    padding: 16,
    display: 'flex',
    overflow: 'hidden',
  },
  idle: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    background: '#ffffff',
    borderRadius: 8,
    border: '1px solid #e2e8f0',
  },
  idleIcon: {
    width: 64,
    height: 64,
    borderRadius: 16,
    background: '#f1f5f9',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: 28,
    marginBottom: 16,
  },
  idleTitle: {
    fontSize: 16,
    fontWeight: 600,
    color: '#64748b',
    marginBottom: 4,
  },
  idleHint: {
    fontSize: 13,
    color: '#94a3b8',
  },
  checking: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    background: '#ffffff',
    borderRadius: 8,
    border: '1px solid #e2e8f0',
  },
  spinner: {
    width: 32,
    height: 32,
    borderRadius: '50%',
    border: '3px solid #e2e8f0',
    borderTopColor: '#3b82f6',
    marginBottom: 16,
    animation: 'spin 0.8s linear infinite',
  },
  checkingText: {
    fontSize: 14,
    color: '#94a3b8',
  },
};

function App() {
  const [state, dispatch] = useReducer(appReducer, initialState);

  return (
    <AppStateContext.Provider value={state}>
      <AppDispatchContext.Provider value={dispatch}>
        <AppInner />
      </AppDispatchContext.Provider>
    </AppStateContext.Provider>
  );
}

function AppInner() {
  useWailsEvents();
  const { phase } = React.useContext(AppStateContext);

  return (
    <div style={STYLE.app}>
      <TitleBar />
      <div style={STYLE.main}>
        <div style={STYLE.sidebar}>
          <DevicePanel />
          <ControlPanel />
          <StatusPanel />
        </div>
        <div style={STYLE.content}>
          {phase === 'checking' && (
            <div style={STYLE.checking}>
              <div style={{ ...STYLE.spinner, animation: 'spin 0.8s linear infinite' }} />
              <div style={STYLE.checkingText}>正在初始化系统...</div>
            </div>
          )}

          {(phase === 'ready') && (
            <div style={STYLE.idle}>
              <div style={STYLE.idleIcon}>🎙️</div>
              <div style={STYLE.idleTitle}>准备就绪</div>
              <div style={STYLE.idleHint}>选择麦克风设备后点击"开始录音"</div>
            </div>
          )}

          {phase === 'recording' && <LiveView />}
          {phase === 'processing' && <ProgressView />}
          {phase === 'result' && <ResultView />}
        </div>
      </div>
    </div>
  );
}

export default App;
