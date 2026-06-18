import React, { useEffect, useRef, useState, useCallback } from 'react';
import { X, Terminal, Loader2, CheckCircle2, XCircle, Trash2 } from 'lucide-react';
import { useUserStore } from '../stores/useUserStore';

interface DeployLogPanelProps {
  projectId: number;
  projectName: string;
  envKey: string;
  routeName: string;
  mode?: 'deploy' | 'stop' | 'restart' | 'logs';
  streamPath?: string;
  panelTitle?: string;
  introText?: string;
  allowCloseWhileRunning?: boolean;
  onClose: () => void;
}

export default function DeployLogPanel({
  projectId,
  projectName,
  envKey,
  routeName,
  mode = 'deploy',
  streamPath,
  panelTitle,
  introText,
  allowCloseWhileRunning = false,
  onClose
}: DeployLogPanelProps) {
  const [logs, setLogs] = useState<string[]>([]);
  const [status, setStatus] = useState<'connecting' | 'running' | 'success' | 'error'>('connecting');
  const [errorMsg, setErrorMsg] = useState('');
  const logEndRef = useRef<HTMLDivElement>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  const scrollToBottom = useCallback(() => {
    logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [logs, scrollToBottom]);

  useEffect(() => {
    const token = useUserStore.getState().token;
    const streamEndpoint = mode === 'logs' ? 'dockerLogStream' : mode === 'stop' ? 'stopStream' : mode === 'restart' ? 'restartStream' : 'deployStream';

    // 判断是否在 Wails 桌面环境中运行
    // Wails 的 AssetServer.Handler 通过 IPC 转发 HTTP 请求，不支持 SSE 流式推送。
    // 因此在 Wails 环境下，EventSource 需要连接到后端启动的真实 HTTP sidecar 服务。
    const isWails = !!(window as any).__wails__ ||
      window.location.protocol === 'wails:' ||
      (window.location.hostname === 'wails.localhost');
    
    let sseBaseUrl: string;
    if (isWails) {
      // Wails 模式: 连接 sidecar 真实 HTTP 服务 (对应 main.go 中 WailsSSEPort)
      sseBaseUrl = 'http://127.0.0.1:48009';
    } else {
      // 浏览器开发模式: 走 Vite proxy 或直连后端
      sseBaseUrl = import.meta.env.VITE_BASE_API || '/api';
    }

    const withToken = (path: string) => {
      const joiner = path.includes('?') ? '&' : '?';
      return `${sseBaseUrl}${path}${joiner}token=${encodeURIComponent(token || '')}`;
    };
    const url = streamPath
      ? withToken(streamPath)
      : `${sseBaseUrl}/project/${streamEndpoint}/${projectId}?env=${encodeURIComponent(envKey)}&token=${encodeURIComponent(token || '')}`;

    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setStatus('running');
      const action = mode === 'logs' ? '读取 Docker 日志' : mode === 'stop' ? '关闭' : mode === 'restart' ? '重启' : '部署';
      setLogs(prev => [...prev, introText || `🚀 已连接，开始${action} [${projectName}] - ${routeName}...`]);
    };

    es.addEventListener('log', (e: MessageEvent) => {
      setLogs(prev => [...prev, e.data]);
    });

    es.addEventListener('done', (e: MessageEvent) => {
      setLogs(prev => [...prev, `\n✅ ${e.data}`]);
      setStatus('success');
      es.close();
    });

    es.addEventListener('error', (e: MessageEvent) => {
      if (e.data) {
        setLogs(prev => [...prev, `\n❌ ${e.data}`]);
        setErrorMsg(e.data);
      }
      setStatus('error');
      es.close();
    });

    es.onerror = () => {
      // EventSource auto error fires when connection closes
      if (es.readyState === EventSource.CLOSED) {
        // Normal close after done/error event
        return;
      }
      setLogs(prev => [...prev, '❌ 连接中断']);
      setStatus('error');
      es.close();
    };

    return () => {
      es.close();
    };
  }, [projectId, envKey, projectName, routeName, mode, streamPath, introText]);

  const isStop = mode === 'stop';
  const isRestart = mode === 'restart';
  const isDockerLogs = mode === 'logs';
  const statusConfig = {
    connecting: { icon: <Loader2 size={14} className="animate-spin" />, text: '连接中...', color: 'text-gray-500' },
    running:    { icon: <Loader2 size={14} className="animate-spin" />, text: isDockerLogs ? '实时跟踪中...' : isStop ? '关闭中...' : isRestart ? '重启中...' : '部署中...', color: isDockerLogs ? 'text-emerald-400' : isStop ? 'text-orange-500' : isRestart ? 'text-amber-500' : 'text-blue-600' },
    success:    { icon: <CheckCircle2 size={14} />, text: isDockerLogs ? '日志已断开' : isStop ? '关闭成功' : isRestart ? '重启成功' : '部署成功', color: 'text-green-600' },
    error:      { icon: <XCircle size={14} />, text: isDockerLogs ? '日志读取失败' : isStop ? '关闭失败' : isRestart ? '重启失败' : '部署失败', color: 'text-red-600' },
  };

  const current = statusConfig[status];

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-gray-900 rounded-2xl shadow-2xl w-[800px] max-w-[90vw] max-h-[80vh] flex flex-col border border-gray-700 animate-in zoom-in-95 duration-300">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3 border-b border-gray-700/50">
          <div className="flex items-center gap-3">
            <div className={`p-1.5 ${isDockerLogs ? 'bg-emerald-500/20 text-emerald-300' : isStop ? 'bg-orange-500/20 text-orange-400' : isRestart ? 'bg-amber-500/20 text-amber-300' : 'bg-green-500/20 text-green-400'} rounded-lg`}>
              <Terminal size={16} />
            </div>
            <div>
              <h3 className="text-sm font-bold text-white leading-tight">{panelTitle || (isDockerLogs ? 'Docker 实时日志' : isStop ? '关闭日志' : isRestart ? '重启日志' : '部署日志')}</h3>
              <p className="text-xs text-gray-400">{projectName} · {routeName}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <span className={`flex items-center gap-1.5 text-xs font-medium ${current.color}`}>
              {current.icon} {current.text}
            </span>
            <button
              onClick={onClose}
              className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
            >
              <X size={16} />
            </button>
          </div>
        </div>

        {/* Log Body */}
        <div className="flex-1 overflow-y-auto px-5 py-4 font-mono text-xs leading-relaxed min-h-[300px] max-h-[60vh]">
          {logs.length === 0 ? (
            <div className="flex items-center justify-center h-full text-gray-500">
              <Loader2 size={18} className="animate-spin mr-2" /> 等待日志输出...
            </div>
          ) : (
            logs.map((line, i) => (
              <div key={i} className="py-0.5">
                <span className="text-gray-500 select-none mr-3">{String(i + 1).padStart(3, ' ')}</span>
                <span className={
                  line.startsWith('✅') ? 'text-green-400' :
                  line.startsWith('❌') ? 'text-red-400' :
                  line.startsWith('📋') || line.startsWith('🚀') || line.startsWith('🔨') || line.startsWith('🐳') || line.startsWith('📦') || line.startsWith('💾') || line.startsWith('🏠') || line.startsWith('🛑') || line.startsWith('⏹️') ? 'text-blue-400' :
                  line.includes('error') || line.includes('ERROR') || line.includes('failed') ? 'text-red-300' :
                  line.includes('warning') || line.includes('WARN') ? 'text-yellow-300' :
                  'text-gray-300'
                }>
                  {line}
                </span>
              </div>
            ))
          )}
          <div ref={logEndRef} />
        </div>

        {/* Footer */}
        <div className="border-t border-gray-700/50 px-5 py-3 flex items-center justify-between">
          <span className="text-xs text-gray-500">{logs.length} 行日志</span>
          <div className="flex gap-2">
            <button
              onClick={() => setLogs([])}
              className="flex items-center gap-1 px-3 py-1.5 text-xs text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
            >
              <Trash2 size={12} /> 清空
            </button>
            <button
              onClick={onClose}
              className={`px-4 py-1.5 text-xs font-medium rounded-lg transition-colors ${
                !allowCloseWhileRunning && !isDockerLogs && (status === 'running' || status === 'connecting')
                  ? 'bg-gray-700 text-gray-400 cursor-not-allowed'
                  : 'bg-white text-gray-900 hover:bg-gray-200'
              }`}
              disabled={!allowCloseWhileRunning && !isDockerLogs && (status === 'running' || status === 'connecting')}
            >
              关闭
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
