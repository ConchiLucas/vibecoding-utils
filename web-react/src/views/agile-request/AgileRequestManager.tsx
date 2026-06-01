import { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import clsx from 'clsx';
import {
  Clock,
  Code2,
  Copy,
  Eraser,
  History,
  Loader2,
  Play,
  RotateCcw,
  Search,
  Trash2,
  Upload,
  X,
} from 'lucide-react';
import {
  AgileMethod,
  AgileRequestLog,
  clearAgileRequestHistory,
  deleteAgileRequestHistory,
  getAgileRequestDetail,
  getAgileRequestHistory,
  sendAgileRequest,
} from '../../api/agileRequest';
import { useConfirm } from '../../hooks/useConfirm';
import ConfirmDialog from '../../components/ConfirmDialog';

const methodOptions: AgileMethod[] = ['GET', 'POST', 'PUT', 'DELETE'];

const defaultHeaders = '{\n  "Content-Type": "application/json"\n}';
const defaultBody = '{\n  \n}';

const tryFormatJson = (value: string) => {
  const text = value.trim();
  if (!text) return '';
  return JSON.stringify(JSON.parse(text), null, 2);
};

const formatMaybeJson = (value: string) => {
  try {
    return tryFormatJson(value);
  } catch {
    return value;
  }
};

const formatDateTime = (value?: string) => {
  if (!value) return '-';
  return new Date(value).toLocaleString();
};

const responseSize = (value: string) => {
  const bytes = new Blob([value || '']).size;
  if (bytes < 1024) return `${bytes} B`;
  return `${(bytes / 1024).toFixed(1)} KB`;
};

type ImportedFetchOptions = {
  headers?: Record<string, unknown>;
  body?: unknown;
  method?: string;
};

const parseFetchImport = (source: string) => {
  const text = source.trim();
  const matched = text.match(/^fetch\(\s*(["'`])([\s\S]*?)\1\s*(?:,\s*([\s\S]*?)\s*)?\)\s*;?\s*$/);
  if (!matched) {
    throw new Error('请粘贴浏览器 Copy as fetch 的完整内容');
  }

  const requestUrl = matched[2].trim();
  const rawOptions = (matched[3] || '').trim();
  if (!rawOptions) {
    return {
      method: 'GET' as AgileMethod,
      url: requestUrl,
      headersJson: '{}',
      body: '',
    };
  }

  let options: ImportedFetchOptions;
  try {
    options = JSON.parse(rawOptions) as ImportedFetchOptions;
  } catch {
    throw new Error('暂只支持 Chrome 默认 Copy as fetch 生成的 JSON options 格式');
  }

  const nextMethod = (options.method || (options.body ? 'POST' : 'GET')).toUpperCase();
  if (!methodOptions.includes(nextMethod as AgileMethod)) {
    throw new Error(`暂不支持导入 ${nextMethod} 请求`);
  }

  let nextBody = '';
  if (typeof options.body === 'string') {
    nextBody = formatMaybeJson(options.body);
  } else if (typeof options.body !== 'undefined' && options.body !== null) {
    nextBody = JSON.stringify(options.body, null, 2);
  }

  return {
    method: nextMethod as AgileMethod,
    url: requestUrl,
    headersJson: JSON.stringify(options.headers || {}, null, 2),
    body: nextBody,
  };
};

export default function AgileRequestManager() {
  const [method, setMethod] = useState<AgileMethod>('POST');
  const [url, setUrl] = useState('');
  const [headersJson, setHeadersJson] = useState(defaultHeaders);
  const [body, setBody] = useState(defaultBody);
  const [activeResponseTab, setActiveResponseTab] = useState<'json' | 'headers' | 'raw'>('json');
  const [history, setHistory] = useState<AgileRequestLog[]>([]);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [importText, setImportText] = useState('');
  const [historyLoading, setHistoryLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [response, setResponse] = useState<AgileRequestLog | null>(null);
  const { confirm, dialogProps } = useConfirm();

  const responseBody = response?.responseBody || '';
  const responseHeaders = useMemo(() => formatMaybeJson(response?.responseHeaders || ''), [response?.responseHeaders]);

  const loadHistory = async (silent = false) => {
    setHistoryLoading(true);
    try {
      const res = await getAgileRequestHistory({ page: 1, pageSize: 50, keyword: keyword.trim() || undefined });
      setHistory(res.data?.list || []);
    } catch {
      if (!silent) {
        toast.error('加载历史失败');
      }
    } finally {
      setHistoryLoading(false);
    }
  };

  useEffect(() => {
    loadHistory(true);
  }, []);

  const validateJsonField = (value: string, label: string) => {
    if (!value.trim()) return '';
    try {
      return tryFormatJson(value);
    } catch {
      toast.error(`${label} JSON 格式不正确`);
      throw new Error(`${label} JSON 格式不正确`);
    }
  };

  const formatJsonField = (value: string, setValue: (next: string) => void, label: string) => {
    try {
      setValue(tryFormatJson(value));
      toast.success(`${label} 已格式化`);
    } catch {
      toast.error(`${label} JSON 格式不正确`);
    }
  };

  const handleSend = async () => {
    const requestUrl = url.trim();
    if (!requestUrl) {
      toast.error('请输入请求 URL');
      return;
    }

    let nextHeaders = '';
    let nextBody = '';
    try {
      nextHeaders = validateJsonField(headersJson, 'Header');
      nextBody = validateJsonField(body, '请求体');
    } catch {
      return;
    }

    setSending(true);
    try {
      const res = await sendAgileRequest({
        method,
        url: requestUrl,
        requestHeaders: nextHeaders || '{}',
        requestBody: nextBody,
      });
      if (res.data) {
        setResponse(res.data);
        setSelectedId(res.data.ID);
        setActiveResponseTab('json');
        await loadHistory();
        if (res.code === 0) {
          toast.success('请求完成');
        }
      }
    } catch {
      toast.error('请求执行失败');
    } finally {
      setSending(false);
    }
  };

  const handleHistoryClick = async (item: AgileRequestLog) => {
    setSelectedId(item.ID);
    try {
      const res = await getAgileRequestDetail({ id: item.ID });
      const detail = res.data;
      setMethod(detail.method);
      setUrl(detail.url);
      setHeadersJson(formatMaybeJson(detail.requestHeaders || '{}'));
      setBody(formatMaybeJson(detail.requestBody));
      setResponse(detail);
      setActiveResponseTab('json');
      setHistoryOpen(false);
    } catch {
      toast.error('加载历史详情失败');
    }
  };

  const handleDeleteHistory = async (item: AgileRequestLog, event: React.MouseEvent) => {
    event.stopPropagation();
    const ok = await confirm(`确定删除这条 ${item.method} 请求历史吗？`);
    if (!ok) return;
    try {
      await deleteAgileRequestHistory(item.ID);
      if (selectedId === item.ID) {
        setSelectedId(null);
      }
      await loadHistory();
      toast.success('历史已删除');
    } catch {
      toast.error('删除失败');
    }
  };

  const handleClearHistory = async () => {
    const ok = await confirm('确定清空敏捷请求历史吗？');
    if (!ok) return;
    try {
      await clearAgileRequestHistory();
      setHistory([]);
      setSelectedId(null);
      toast.success('历史已清空');
    } catch {
      toast.error('清空失败');
    }
  };

  const resetRequest = () => {
    setMethod('POST');
    setUrl('');
    setHeadersJson(defaultHeaders);
    setBody(defaultBody);
    setResponse(null);
    setSelectedId(null);
  };

  const copyResponse = async () => {
    if (!responseBody) return;
    await navigator.clipboard.writeText(responseBody);
    toast.success('响应已复制');
  };

  const handleImportFetch = () => {
    try {
      const imported = parseFetchImport(importText);
      setMethod(imported.method);
      setUrl(imported.url);
      setHeadersJson(imported.headersJson || '{}');
      setBody(imported.body || '');
      setResponse(null);
      setSelectedId(null);
      setImportOpen(false);
      toast.success('请求已导入');
    } catch (error: any) {
      toast.error(error?.message || '导入失败');
    }
  };

  return (
    <div className="h-[calc(100vh-64px)] min-h-[720px] bg-white flex flex-col overflow-hidden">
      <div className="h-16 px-5 border-b border-slate-200 flex items-center gap-3">
        <select
          value={method}
          onChange={e => setMethod(e.target.value as AgileMethod)}
          className="h-10 w-28 rounded-lg border border-slate-300 bg-white px-3 text-sm font-bold outline-none focus:ring-2 focus:ring-slate-900/10"
        >
          {methodOptions.map(item => <option key={item} value={item}>{item}</option>)}
        </select>
        <input
          value={url}
          onChange={e => setUrl(e.target.value)}
          className="h-10 flex-1 rounded-lg border border-slate-300 px-3 text-sm font-mono outline-none focus:ring-2 focus:ring-slate-900/10"
          placeholder="https://api.example.com/resource"
        />
        <button
          type="button"
          onClick={() => {
            setHistoryOpen(true);
            loadHistory();
          }}
          className="h-10 px-3 inline-flex items-center gap-2 rounded-lg border border-slate-200 text-slate-700 text-sm font-medium hover:bg-slate-50 transition"
        >
          <History size={17} />
          历史
        </button>
        <button
          type="button"
          onClick={() => setImportOpen(true)}
          className="h-10 px-3 inline-flex items-center gap-2 rounded-lg border border-slate-200 text-slate-700 text-sm font-medium hover:bg-slate-50 transition"
        >
          <Upload size={17} />
          导入
        </button>
        <button
          type="button"
          onClick={resetRequest}
          className="h-10 w-10 inline-flex items-center justify-center rounded-lg border border-slate-200 text-slate-500 hover:bg-slate-50 transition"
          title="重置"
        >
          <RotateCcw size={17} />
        </button>
        <button
          type="button"
          onClick={handleSend}
          disabled={sending}
          className="h-10 px-5 inline-flex items-center gap-2 rounded-lg bg-slate-950 text-white text-sm font-semibold hover:bg-slate-800 disabled:opacity-50 transition"
        >
          {sending ? <Loader2 size={17} className="animate-spin" /> : <Play size={17} fill="currentColor" />}
          发送
        </button>
      </div>

      <div className="flex-1 min-h-0 flex">
        <section className="w-[58%] min-w-[520px] flex flex-col border-r border-slate-200 min-h-0">
          <div className="flex-1 min-h-0 flex flex-col border-b border-slate-200">
            <div className="h-11 px-5 border-b border-slate-100 flex items-center justify-between bg-slate-50">
              <span className="inline-flex items-center gap-2 text-sm font-semibold text-slate-700">
                <Code2 size={16} />
                Header JSON
              </span>
              <button
                type="button"
                onClick={() => formatJsonField(headersJson, setHeadersJson, 'Header')}
                className="h-8 px-3 rounded-lg text-sm font-medium text-slate-600 hover:bg-slate-100 transition"
              >
                格式化
              </button>
            </div>
            <textarea
              value={headersJson}
              onChange={e => setHeadersJson(e.target.value)}
              spellCheck={false}
              className="flex-1 w-full resize-none outline-none p-5 font-mono text-sm text-slate-800 leading-6"
              placeholder='{"Content-Type":"application/json"}'
            />
          </div>

          <div className="flex-1 min-h-0 flex flex-col">
            <div className="h-11 px-5 border-b border-slate-100 flex items-center justify-between bg-slate-50">
              <span className="inline-flex items-center gap-2 text-sm font-semibold text-slate-700">
                <Code2 size={16} />
                请求 JSON
              </span>
              <button
                type="button"
                onClick={() => formatJsonField(body, setBody, '请求体')}
                className="h-8 px-3 rounded-lg text-sm font-medium text-slate-600 hover:bg-slate-100 transition"
              >
                格式化
              </button>
            </div>
            <textarea
              value={body}
              onChange={e => setBody(e.target.value)}
              spellCheck={false}
              className="flex-1 w-full resize-none outline-none p-5 font-mono text-sm text-slate-800 leading-6"
              placeholder="{ }"
            />
          </div>
        </section>

        <section className="flex-1 min-w-[420px] flex flex-col bg-slate-950 text-slate-100">
          <div className="h-16 px-5 border-b border-white/10 flex items-center justify-between">
            <div>
              <div className="text-sm font-bold">Response</div>
              <div className="mt-1 flex items-center gap-3 text-xs text-slate-400">
                <span className={clsx(response?.isSuccess ? 'text-emerald-400' : 'text-red-400')}>
                  {response ? response.responseStatus || 'ERR' : '-'}
                </span>
                <span>{response ? `${response.durationMs}ms` : '-'}</span>
                <span>{response ? responseSize(responseBody) : '-'}</span>
              </div>
            </div>
            <button
              type="button"
              onClick={copyResponse}
              disabled={!responseBody}
              className="h-9 w-9 inline-flex items-center justify-center rounded-lg text-slate-300 hover:bg-white/10 disabled:opacity-30 transition"
              title="复制响应"
            >
              <Copy size={16} />
            </button>
          </div>

          <div className="h-12 px-5 border-b border-white/10 flex items-center gap-2">
            {(['json', 'headers', 'raw'] as const).map(tab => (
              <button
                key={tab}
                type="button"
                onClick={() => setActiveResponseTab(tab)}
                className={clsx(
                  'h-8 px-3 rounded-lg text-sm font-medium transition capitalize',
                  activeResponseTab === tab ? 'bg-white/10 text-white' : 'text-slate-400 hover:bg-white/5'
                )}
              >
                {tab}
              </button>
            ))}
          </div>

          <div className="flex-1 min-h-0 relative">
            <textarea
              value={activeResponseTab === 'headers' ? responseHeaders : responseBody}
              readOnly
              spellCheck={false}
              className="absolute inset-0 w-full h-full resize-none outline-none p-5 bg-transparent font-mono text-sm leading-6 text-emerald-300"
              placeholder="等待响应..."
            />
          </div>
        </section>
      </div>

      {historyOpen && (
        <div className="fixed inset-0 z-[80] bg-slate-950/40 backdrop-blur-sm flex items-center justify-center p-6">
          <div className="w-full max-w-3xl max-h-[78vh] bg-white rounded-xl shadow-2xl border border-slate-200 flex flex-col overflow-hidden">
            <div className="h-14 px-5 border-b border-slate-200 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <History size={18} className="text-slate-700" />
                <span className="text-sm font-bold text-slate-900">历史请求</span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={handleClearHistory}
                  className="h-9 px-3 inline-flex items-center gap-2 rounded-lg text-sm text-slate-500 hover:text-red-600 hover:bg-red-50 transition"
                >
                  <Eraser size={16} />
                  清空
                </button>
                <button
                  type="button"
                  onClick={() => setHistoryOpen(false)}
                  className="h-9 w-9 inline-flex items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 transition"
                  title="关闭"
                >
                  <X size={17} />
                </button>
              </div>
            </div>

            <form
              onSubmit={(event) => {
                event.preventDefault();
                loadHistory();
              }}
              className="p-4 border-b border-slate-200"
            >
              <div className="relative">
                <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                <input
                  value={keyword}
                  onChange={e => setKeyword(e.target.value)}
                  className="w-full h-10 pl-9 pr-3 rounded-lg border border-slate-200 bg-white text-sm outline-none focus:ring-2 focus:ring-slate-900/10"
                  placeholder="搜索 URL / Body"
                />
              </div>
            </form>

            <div className="flex-1 min-h-0 overflow-y-auto">
              {historyLoading ? (
                <div className="h-40 flex items-center justify-center text-slate-400">
                  <Loader2 size={18} className="animate-spin" />
                </div>
              ) : history.length === 0 ? (
                <div className="px-5 py-12 text-center text-sm text-slate-400">暂无历史</div>
              ) : (
                history.map(item => (
                  <button
                    key={item.ID}
                    type="button"
                    onClick={() => handleHistoryClick(item)}
                    className={clsx(
                      'w-full text-left px-5 py-4 border-b border-slate-100 transition group',
                      selectedId === item.ID ? 'bg-slate-50' : 'hover:bg-slate-50'
                    )}
                  >
                    <div className="flex items-center gap-3">
                      <span className={clsx(
                        'w-16 text-xs font-bold',
                        item.method === 'GET' && 'text-blue-600',
                        item.method === 'POST' && 'text-emerald-600',
                        item.method === 'PUT' && 'text-amber-600',
                        item.method === 'DELETE' && 'text-red-600'
                      )}>{item.method}</span>
                      <span className={clsx(
                        'text-xs font-semibold',
                        item.isSuccess ? 'text-emerald-600' : 'text-red-500'
                      )}>{item.responseStatus || 'ERR'}</span>
                      <span className="text-xs text-slate-400">{item.durationMs}ms</span>
                      <span className="ml-auto inline-flex items-center gap-1 text-xs text-slate-400">
                        <Clock size={12} />
                        {formatDateTime(item.CreatedAt)}
                      </span>
                      <button
                        type="button"
                        onClick={(event) => handleDeleteHistory(item, event)}
                        className="opacity-0 group-hover:opacity-100 p-1.5 rounded text-slate-400 hover:text-red-600 hover:bg-red-50 transition"
                        title="删除"
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                    <div className="mt-2 text-sm text-slate-700 truncate">{item.url}</div>
                  </button>
                ))
              )}
            </div>
          </div>
        </div>
      )}

      {importOpen && (
        <div className="fixed inset-0 z-[80] bg-slate-950/40 backdrop-blur-sm flex items-center justify-center p-6">
          <div className="w-full max-w-3xl h-[68vh] bg-white rounded-xl shadow-2xl border border-slate-200 flex flex-col overflow-hidden">
            <div className="h-14 px-5 border-b border-slate-200 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Upload size={18} className="text-slate-700" />
                <span className="text-sm font-bold text-slate-900">导入 Copy as fetch</span>
              </div>
              <button
                type="button"
                onClick={() => setImportOpen(false)}
                className="h-9 w-9 inline-flex items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 transition"
                title="关闭"
              >
                <X size={17} />
              </button>
            </div>
            <textarea
              value={importText}
              onChange={e => setImportText(e.target.value)}
              spellCheck={false}
              className="flex-1 min-h-0 resize-none outline-none p-5 font-mono text-sm text-slate-800 leading-6"
              placeholder='fetch("http://localhost:5175/api/server/buildTree", { ... });'
            />
            <div className="h-16 px-5 border-t border-slate-200 flex items-center justify-between">
              <button
                type="button"
                onClick={() => setImportText('')}
                className="h-10 px-3 rounded-lg text-sm text-slate-500 hover:bg-slate-100 transition"
              >
                清空
              </button>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setImportOpen(false)}
                  className="h-10 px-4 rounded-lg border border-slate-200 text-sm font-medium text-slate-700 hover:bg-slate-50 transition"
                >
                  取消
                </button>
                <button
                  type="button"
                  onClick={handleImportFetch}
                  className="h-10 px-5 rounded-lg bg-slate-950 text-white text-sm font-semibold hover:bg-slate-800 transition"
                >
                  导入请求
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
