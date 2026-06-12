// =============================================================================
// MyMeetily — ResultView (会议纪要展示)
// =============================================================================

import React, { useState } from 'react';
import { useAppState, useAppDispatch } from '../hooks/useAppState';
import { AppService, PipelineService } from '../hooks/useWailsEvents';

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
    borderBottom: '1px solid #e2e8f0',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  title: {
    fontSize: 15,
    fontWeight: 600,
    color: '#1e293b',
  },
  stats: {
    fontSize: 12,
    color: '#64748b',
    marginTop: 2,
  },
  actions: {
    display: 'flex',
    gap: 8,
  },
  actionBtn: {
    padding: '6px 14px',
    borderRadius: 6,
    border: '1px solid #e2e8f0',
    background: '#f8fafc',
    fontSize: 12,
    color: '#475569',
    cursor: 'pointer',
    fontWeight: 500,
  },
  primaryBtn: {
    padding: '6px 14px',
    borderRadius: 6,
    border: 'none',
    background: '#3b82f6',
    fontSize: 12,
    color: '#ffffff',
    cursor: 'pointer',
    fontWeight: 500,
  },
  body: {
    flex: 1,
    overflow: 'auto' as const,
    padding: 20,
  },
  notes: {
    fontSize: 14,
    color: '#334155',
    lineHeight: 1.8,
  },
  // Simple meeting notes styles
  noteH1: {
    fontSize: 20,
    fontWeight: 700,
    color: '#1e293b',
    marginTop: 20,
    marginBottom: 10,
  },
  noteH2: {
    fontSize: 16,
    fontWeight: 600,
    color: '#1e293b',
    marginTop: 16,
    marginBottom: 8,
    paddingBottom: 6,
    borderBottom: '1px solid #f1f5f9',
  },
  noteH3: {
    fontSize: 14,
    fontWeight: 600,
    color: '#475569',
    marginTop: 12,
    marginBottom: 6,
  },
  noteBlockquote: {
    borderLeft: '3px solid #3b82f6',
    paddingLeft: 12,
    margin: '8px 0',
    color: '#64748b',
  },
  noteUl: {
    paddingLeft: 20,
    marginTop: 4,
    marginBottom: 8,
  },
  noteLi: {
    marginBottom: 4,
  },
  error: {
    padding: '12px 16px',
    background: '#fef2f2',
    border: '1px solid #fecaca',
    borderRadius: 6,
    color: '#dc2626',
    fontSize: 13,
    marginTop: 8,
  },
  status: {
    fontSize: 12,
    padding: '4px 10px',
    borderRadius: 4,
    marginTop: 8,
  },
  files: {
    marginTop: 16,
    padding: 12,
    background: '#f8fafc',
    borderRadius: 6,
    border: '1px solid #e2e8f0',
  },
  fileTitle: {
    fontSize: 12,
    fontWeight: 600,
    color: '#64748b',
    marginBottom: 8,
  },
  fileItem: {
    fontSize: 12,
    color: '#475569',
    padding: '2px 0',
    fontFamily: "'SF Mono', 'Fira Code', monospace",
  },
};

function renderSimpleMarkdown(markdown: string): React.ReactNode {
  if (!markdown) return null;

  const lines = markdown.split('\n');
  const elements: React.ReactNode[] = [];
  let inList = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    if (line === '') {
      inList = false;
      continue;
    }

    if (line.startsWith('# ')) {
      inList = false;
      elements.push(
        <div key={i} style={STYLE.noteH1}>{line.slice(2)}</div>
      );
    } else if (line.startsWith('## ')) {
      inList = false;
      elements.push(
        <div key={i} style={STYLE.noteH2}>{line.slice(3)}</div>
      );
    } else if (line.startsWith('### ')) {
      inList = false;
      elements.push(
        <div key={i} style={STYLE.noteH3}>{line.slice(4)}</div>
      );
    } else if (line.startsWith('> ')) {
      inList = false;
      elements.push(
        <div key={i} style={STYLE.noteBlockquote}>{line.slice(2)}</div>
      );
    } else if (line.startsWith('- ') || line.startsWith('* ')) {
      if (!inList) {
        elements.push(<ul key={`ul-${i}`} style={STYLE.noteUl} />);
        inList = true;
      }
      // Don't actually append here, this is simplified
      elements.push(
        <div key={i} style={{ paddingLeft: 16, color: '#334155', fontSize: 14, lineHeight: 1.7 }}>
          • {line.replace(/^[-*]\s+/, '')}
        </div>
      );
    } else if (line.startsWith('---')) {
      inList = false;
      elements.push(
        <hr key={i} style={{ border: 'none', borderTop: '1px solid #e2e8f0', margin: '12px 0' }} />
      );
    } else {
      inList = false;
      // Bold inline
      const parts = line.split(/(\*\*[^*]+\*\*)/g);
      const rendered = parts.map((part, j) => {
        if (part.startsWith('**') && part.endsWith('**')) {
          return <strong key={j}>{part.slice(2, -2)}</strong>;
        }
        return part;
      });
      elements.push(
        <p key={i} style={{ margin: '4px 0', fontSize: 14, color: '#334155', lineHeight: 1.7 }}>
          {rendered}
        </p>
      );
    }
  }

  return elements;
}

export const ResultView: React.FC = () => {
  const { meetingState, actionStatus, actionError } = useAppState();
  const dispatch = useAppDispatch();
  const [retrying, setRetrying] = useState(false);

  if (!meetingState) return null;

  const handleCopySummary = async () => {
    try {
      await AppService().CopyToClipboard(meetingState.meetingNotes || meetingState.summaryContent);
      dispatch({ type: 'SET_ACTION_STATUS', status: '已复制会议纪要', error: false });
    } catch {
      dispatch({ type: 'SET_ACTION_STATUS', status: '复制失败', error: true });
    }
  };

  const handleCopyTranscript = async () => {
    try {
      await AppService().CopyToClipboard(meetingState.rawTranscript);
      dispatch({ type: 'SET_ACTION_STATUS', status: '已复制转写文本', error: false });
    } catch {
      dispatch({ type: 'SET_ACTION_STATUS', status: '复制失败', error: true });
    }
  };

  const handleOpenFolder = async () => {
    try {
      await AppService().OpenOutputFolder(meetingState.htmlOutputPath || meetingState.outputPath);
      dispatch({ type: 'SET_ACTION_STATUS', status: '已打开输出目录', error: false });
    } catch {
      dispatch({ type: 'SET_ACTION_STATUS', status: '打开失败', error: true });
    }
  };

  const handleRetry = async () => {
    setRetrying(true);
    try {
      await PipelineService().RetrySummary();
      dispatch({ type: 'SET_ACTION_STATUS', status: '摘要已重新生成', error: false });
    } catch {
      dispatch({ type: 'SET_ACTION_STATUS', status: '重试失败', error: true });
    } finally {
      setRetrying(false);
    }
  };

  const handleNewRecording = () => {
    dispatch({ type: 'RESET' });
  };

  const formatDuration = (sec: number) => {
    const m = Math.floor(sec / 60);
    const s = Math.floor(sec % 60);
    return `${m}分${s}秒`;
  };

  const content = meetingState.meetingNotes || meetingState.summaryContent;

  return (
    <div style={STYLE.container}>
      <div style={STYLE.header}>
        <div>
          <div style={STYLE.title}>会议纪要</div>
          <div style={STYLE.stats}>
            时长 {formatDuration(meetingState.audioDuration)} | 语言 {meetingState.language} | 转写 {meetingState.rawTranscript?.length || 0} 字
          </div>
        </div>
        <div style={STYLE.actions}>
          <button style={STYLE.actionBtn} onClick={handleCopySummary} title="复制纪要">📋 纪要</button>
          <button style={STYLE.actionBtn} onClick={handleCopyTranscript} title="复制转写">📝 转写</button>
          <button style={STYLE.actionBtn} onClick={handleOpenFolder} title="打开目录">📂</button>
          {(!meetingState.summaryEnabled || meetingState.summaryError) && (
            <button style={STYLE.actionBtn} onClick={handleRetry} disabled={retrying}>
              {retrying ? '重试中...' : '🔄 重试'}
            </button>
          )}
          <button style={STYLE.primaryBtn} onClick={handleNewRecording}>新建录音</button>
        </div>
      </div>

      {meetingState.summaryError && (
        <div style={{ ...STYLE.error, margin: '0 20px', marginTop: 12 }}>
          摘要生成失败: {meetingState.summaryError}
        </div>
      )}

      {actionStatus && (
        <div style={{
          ...STYLE.status,
          color: actionError ? '#dc2626' : '#10b981',
          background: actionError ? '#fef2f2' : '#f0fdf4',
          margin: '0 20px',
          marginTop: 8,
        }}>
          {actionStatus}
        </div>
      )}

      <div style={STYLE.body}>
        <div style={STYLE.notes}>
          {renderSimpleMarkdown(content)}
        </div>

        <div style={STYLE.files}>
          <div style={STYLE.fileTitle}>已生成文件</div>
          {meetingState.htmlOutputPath && (
            <div style={STYLE.fileItem}>📄 HTML: {meetingState.htmlOutputPath}</div>
          )}
          {meetingState.markdownPath && (
            <div style={STYLE.fileItem}>📝 Markdown: {meetingState.markdownPath}</div>
          )}
          {meetingState.transcriptPath && (
            <div style={STYLE.fileItem}>📃 转写: {meetingState.transcriptPath}</div>
          )}
        </div>
      </div>
    </div>
  );
};
