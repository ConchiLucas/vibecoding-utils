import React, { useEffect, useState } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { getScriptPage, deleteScript, previewScriptFile, saveOrUpdateScript } from '../../api/script';
import { getProjectById } from '../../api/project';
import Editor from '@monaco-editor/react';
import toast from 'react-hot-toast';
import { FileCode, File, Trash2, ArrowLeft, Save, RefreshCw, Plus } from 'lucide-react';
import clsx from 'clsx';

export default function ProjectScripts() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const query = new URLSearchParams(location.search);
  const routeIdParam = query.get('routeId');
  const routeId = routeIdParam ? parseInt(routeIdParam) : 0;

  const [projectInfo, setProjectInfo] = useState<any>(null);
  const [scripts, setScripts] = useState<any[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [loadingList, setLoadingList] = useState(false);

  const [activeScript, setActiveScript] = useState<any>(null);
  const [codeContent, setCodeContent] = useState('');
  const [fileNameInput, setFileNameInput] = useState('');
  const [loadingContent, setLoadingContent] = useState(false);
  const [saving, setSaving] = useState(false);
  const [scriptToDelete, setScriptToDelete] = useState<any>(null);

  const fetchProjectInfo = async () => {
    try {
      const res: any = await getProjectById(projectId as string);
      if (res.code === 0) setProjectInfo(res.data);
    } catch (e) {
      console.warn("Could not fetch project info");
    }
  };

  const fetchScripts = async () => {
    setLoadingList(true);
    try {
      const res: any = await getScriptPage({ page: 1, pageSize: 100, projectId: parseInt(projectId as string), routeId });
      if (res.code === 0) {
        const list = res.data?.list || [];
        setScripts(list);
        return list;
      }
    } catch (err) {
      toast.error('拉取脚本列表失败');
    } finally {
      setLoadingList(false);
    }
    return [];
  };

  const startNewScript = () => {
    const dummy = { ID: 0, projectId: parseInt(projectId as string), routeId: routeId, fileName: 'untitled.sh', scriptType: 0, content: '' };
    setActiveScript(dummy);
    setFileNameInput('untitled.sh');
    setCodeContent('');
  };

  useEffect(() => {
    if (projectId) {
      fetchProjectInfo();
      fetchScripts();
    }
  }, [projectId]);

  useEffect(() => {
    // 自动选中第一个脚本（初始加载，或唯一高亮项被删除后）
    if (scripts.length > 0 && !activeScript) {
      handleSelectScript(scripts[0]);
    }
  }, [scripts, activeScript]);

  useEffect(() => {
    const handleNewScript = () => {
      startNewScript();
    };

    const handleSearch = (e: any) => {
      setSearchTerm(e.detail || '');
    };

    window.addEventListener('NEW_SCRIPT', handleNewScript);
    window.addEventListener('SEARCH_SCRIPT', handleSearch);

    return () => {
      window.removeEventListener('NEW_SCRIPT', handleNewScript);
      window.removeEventListener('SEARCH_SCRIPT', handleSearch);
    };
  }, [projectId, routeId]);

  const handleSelectScript = async (script: any) => {
    setActiveScript(script);
    setFileNameInput(script.fileName);
    setLoadingContent(true);
    try {
      const res: any = await previewScriptFile(script.ID);
      if (res.code === 0) {
        setCodeContent(res.data || '');
      } else {
        toast.error('读取内容失败');
        setCodeContent('');
      }
    } catch (err) {
      toast.error('网络读取异常');
      setCodeContent('');
    } finally {
      setLoadingContent(false);
    }
  };

  const handleDeleteClick = (script: any, e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    setScriptToDelete(script);
  };

  const executeDelete = async (script: any) => {
    try {
      const res: any = await deleteScript(script.ID);
      if (res.code === 0) {
        toast.success('已删除');
        if (activeScript?.ID === script.ID) {
          setActiveScript(null);
          setFileNameInput('');
          setCodeContent('');
        }
        fetchScripts();
      }
    } catch (err) {
      toast.error('删除引发异常');
    }
  };

  const handleSaveContent = async () => {
    if (!activeScript) return;
    if (!fileNameInput.trim()) {
      toast.error('必须指定文件名');
      return;
    }
    setSaving(true);
    try {
      const payload = {
        ID: activeScript.ID || 0,
        projectId: parseInt(projectId as string),
        routeId: routeId,
        fileName: fileNameInput.trim(),
        scriptType: activeScript.scriptType || 0,
        content: codeContent
      };
      const res: any = await saveOrUpdateScript(payload);
      if (res.code === 0) {
        toast.success(activeScript.ID ? '修改已被安全覆盖落盘' : '新脚本已创立');
        // Fetch scripts and re-select properly if it was new
        const activeList = await fetchScripts();
        if (activeScript.ID === 0) {
           const savedScript = activeList.find((s: any) => s.fileName === fileNameInput.trim());
           if (savedScript) {
             setActiveScript(savedScript);
           } else {
             setActiveScript({...activeScript, fileName: fileNameInput.trim()});
           }
        }
      } else {
        toast.error(res.msg || '写入失败');
      }
    } catch (err) {
      toast.error('网络操作中断');
    } finally {
      setSaving(false);
    }
  };

  // Determine language natively mapped to monaco
  const getLanguageType = (filename: string) => {
    const fn = (filename || '').toLowerCase();
    if (fn.endsWith('.json')) return 'json';
    if (fn.endsWith('.sh') || fn.endsWith('.bash')) return 'shell';
    if (fn.endsWith('.py')) return 'python';
    if (fn.endsWith('.js')) return 'javascript';
    if (fn.endsWith('.yaml') || fn.endsWith('.yml')) return 'yaml';
    return 'plaintext';
  };

  const filteredScripts = scripts.filter(s => {
    if (!searchTerm) return true;
    const lowerQ = searchTerm.toLowerCase();
    return s.fileName.toLowerCase().includes(lowerQ) || (s.content && s.content.includes(searchTerm));
  });

  return (
    <div className="flex w-full h-[calc(100vh-140px)] border border-gray-200 rounded-xl overflow-hidden bg-white shadow-sm mt-2 animate-in fade-in duration-300">
      
      {/* Left Sidebar: Scripts Explorer */}
      <div className="w-80 border-r border-gray-200 flex flex-col bg-gray-50 flex-shrink-0">
         <div className="p-4 border-b border-gray-200 bg-white">
           <div className="flex items-start justify-between gap-3">
             <div className="min-w-0">
               <h2 className="text-lg font-bold text-gray-900 truncate" title={projectInfo?.projectName || '文件列表'}>
                 {projectInfo?.projectName || '资源加载中...'}
               </h2>
               <p className="text-xs text-gray-500 font-mono mt-1">文件管理器</p>
             </div>
             <button
               type="button"
               onClick={startNewScript}
               className="inline-flex flex-shrink-0 items-center gap-1 rounded-lg bg-gray-900 px-3 py-1.5 text-xs font-semibold text-white shadow-sm transition hover:bg-gray-700"
             >
               <Plus size={13} />
               新建
             </button>
           </div>
         </div>

         {/* File List */}
         <div className="flex-1 overflow-y-auto p-3 space-y-1">
           {loadingList ? (
             <div className="text-sm text-gray-400 text-center mt-10">加载文件数库...</div>
           ) : filteredScripts.length === 0 ? (
             <div className="text-sm text-gray-400 text-center mt-10">
               {searchTerm ? '无匹配脚本' : '尚无脚本当或部署物'}
             </div>
           ) : (
             filteredScripts.map(script => (
               <div 
                 key={script.ID}
                 onClick={() => handleSelectScript(script)}
                 className={clsx(
                   "group flex items-center justify-between p-2 rounded-lg cursor-pointer transition-colors text-sm",
                   activeScript?.ID === script.ID 
                     ? "bg-blue-50 text-blue-700" 
                     : "text-gray-700 hover:bg-gray-100"
                 )}
               >
                 <div className="flex items-center gap-2 overflow-hidden">
                    <File size={16} className={clsx("flex-shrink-0", activeScript?.ID === script.ID ? "text-blue-500" : "text-gray-400")} />
                    <span className="truncate" title={script.fileName}>{script.fileName}</span>
                 </div>
                 <button 
                   onClick={(e) => handleDeleteClick(script, e)}
                   className="flex-shrink-0 text-gray-300 hover:text-red-500 p-1 rounded transition-all"
                   title="删除"
                 >
                   <Trash2 size={14} />
                 </button>
               </div>
             ))
           )}
         </div>
      </div>

      {/* Right Content: Editor */}
      <div className="flex-1 flex flex-col bg-[#1e1e1e] min-w-0 overflow-hidden">
         {activeScript ? (
           <>
             {/* Editor Header */}
             <div className="h-12 border-b border-[#333] flex items-center px-4 bg-[#252526] text-gray-300 select-none gap-3 overflow-hidden">
                  <FileCode size={16} className="text-[#569cd6] flex-shrink-0" />
                  <input 
                    type="text" 
                    value={fileNameInput}
                    onChange={(e) => setFileNameInput(e.target.value)}
                    className="bg-[#3c3c3c] border border-transparent focus:border-[#007fd4] hover:bg-[#4d4d4d] focus:bg-[#3c3c3c] text-sm text-gray-200 outline-none px-2 py-0.5 rounded font-mono flex-1 min-w-[80px] transition-all"
                    placeholder="文件名称"
                  />
                <button 
                  onClick={handleSaveContent}
                  disabled={saving || loadingContent}
                  className="flex-shrink-0 bg-[#0e639c] hover:bg-[#1177bb] disabled:opacity-50 text-white px-3 py-1.5 rounded text-xs font-medium inline-flex items-center gap-1.5 transition-colors whitespace-nowrap"
                >
                  {saving ? <RefreshCw size={14} className="animate-spin" /> : <Save size={14} />}
                  保存
                </button>
             </div>
             
             {/* Monaco Canvas */}
             <div className="flex-1 relative">
               {loadingContent && (
                 <div className="absolute inset-0 z-10 flex items-center justify-center bg-[#1e1e1e]/80">
                   <span className="text-gray-400 text-sm">解析文件中...</span>
                 </div>
               )}
               <Editor
                  height="100%"
                  theme="vs-dark"
                  language={getLanguageType(fileNameInput)}
                  value={codeContent}
                  onChange={(val) => setCodeContent(val || '')}
                  options={{
                    minimap: { enabled: false },
                    fontSize: 14,
                    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
                    wordWrap: "on",
                    padding: { top: 16 }
                  }}
               />
             </div>
           </>
         ) : (
           <div className="flex-1 flex flex-col items-center justify-center text-gray-500">
             <FileCode size={48} className="mb-4 opacity-20" />
             <p>点击“新建脚本”或从左侧树挑选文件以启动代码流</p>
             <button
               type="button"
               onClick={startNewScript}
               className="mt-5 inline-flex items-center gap-2 rounded-xl bg-[#0e639c] px-4 py-2 text-sm font-semibold text-white shadow-lg shadow-black/20 transition hover:bg-[#1177bb]"
             >
               <Plus size={16} />
               新建脚本
             </button>
           </div>
         )}
      </div>

      {scriptToDelete && (
        <div className="fixed inset-0 z-[999] flex items-center justify-center bg-black/40 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-xl shadow-2xl p-6 w-[400px] border border-gray-100 scale-100 animate-in zoom-in-95 duration-200">
            <h3 className="text-lg font-bold text-gray-900 mb-2">确认删除拦截</h3>
            <p className="text-sm text-gray-600 mb-6 leading-relaxed">
              确定要永久移除 <span className="font-mono text-red-500 font-semibold">{scriptToDelete.fileName}</span> 吗？<br/>
              <span className="text-red-400 text-xs shadow-sm mt-1 inline-block">⚠️ 警告：这可能会导致正在依赖该脚本的部署或流水线任务发生致命错误。</span>
            </p>
            <div className="flex justify-end gap-3 mt-4">
              <button 
                onClick={() => setScriptToDelete(null)}
                className="px-4 py-2 text-sm text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors font-medium"
              >
                取消移除
              </button>
              <button 
                onClick={() => {
                  executeDelete(scriptToDelete);
                  setScriptToDelete(null);
                }}
                className="px-4 py-2 text-sm text-white bg-red-500 hover:bg-red-600 rounded-lg transition-colors font-medium shadow-sm flex items-center gap-1"
              >
                <Trash2 size={14} />
                强制删除
              </button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
