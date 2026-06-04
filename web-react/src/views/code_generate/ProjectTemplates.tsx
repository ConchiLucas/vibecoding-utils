import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getPathList, getModelListByPathId, createModel, updateModel } from '../../api/path_model';
import { getProjectList } from '../../api/project';
import Editor from '@monaco-editor/react';
import toast from 'react-hot-toast';
import { FileCode, File, ArrowLeft, Save, RefreshCw, Folder, ChevronDown, ChevronRight } from 'lucide-react';
import clsx from 'clsx';

const unwrapResponseData = (res: any) => {
  return res?.data?.data ?? res?.data ?? [];
};

export default function ProjectTemplates() {
  const { projectId } = useParams();
  const navigate = useNavigate();

  const [projectName, setProjectName] = useState<string>('解析中...');
  const [paths, setPaths] = useState<any[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [loadingList, setLoadingList] = useState(false);

  // Selected state
  const [activePath, setActivePath] = useState<any>(null);
  
  // Model state associated with the activePath
  const [activeModel, setActiveModel] = useState<any>(null);
  const [codeContent, setCodeContent] = useState('');
  const [loadingContent, setLoadingContent] = useState(false);
  const [saving, setSaving] = useState(false);

  const fetchEnv = async () => {
    try {
      // Find the project name
      const pId = parseInt(projectId as string);
      const resProj: any = await getProjectList();
      const projects = unwrapResponseData(resProj);
      if (Array.isArray(projects)) {
        const found = projects.find((p: any) => p.ID === pId);
        if (found) setProjectName(found.projectName);
      }

      setLoadingList(true);
      const resPath: any = await getPathList(pId);
      
      // Filter out those where projectId == pId (Wait, our backend GET /tbgenerateprojectpath/getTbGenerateProjectPathList might ignore our custom filter logic if we didn't add it in the Go code. So let's filter in frontend just in case!)
      let allPaths = unwrapResponseData(resPath);
      if (!Array.isArray(allPaths)) allPaths = [];
      if (allPaths.length > 0 && allPaths[0].projectId !== undefined) {
         allPaths = allPaths.filter((x: any) => x.projectId === pId);
      }
      setPaths(allPaths);
      
      if (allPaths.length > 0 && !activePath) {
        await handleSelectPath(allPaths[0]);
      }
    } catch (e) {
      toast.error("加载项目底层路径表失败");
    } finally {
      setLoadingList(false);
    }
  };

  useEffect(() => {
    if (projectId) {
      fetchEnv();
    }
  }, [projectId]);

  const handleSelectPath = async (pathObj: any) => {
    setActivePath(pathObj);
    setLoadingContent(true);
    setCodeContent('');
    setActiveModel(null);
    try {
      // For this path, find its model
      const res: any = await getModelListByPathId(pathObj.ID);
      let models = unwrapResponseData(res);
      if (!Array.isArray(models)) models = [];
      // Frontend filter map
      models = models.filter((x: any) => x.pathId === pathObj.ID);
      
      if (models.length > 0) {
        setActiveModel(models[0]);
        setCodeContent(models[0].content || '');
      } else {
        // No underlying model created yet for this metadata path
        setCodeContent(''); 
      }
    } catch (err) {
      toast.error('网络读取异常');
    } finally {
      setLoadingContent(false);
    }
  };

  const handleSaveContent = async () => {
    if (!activePath) return;
    setSaving(true);
    try {
      if (activeModel && activeModel.ID) {
        // Execute Update
        const payload = { ...activeModel, content: codeContent };
        await updateModel(payload);
        setActiveModel(payload);
        toast.success("代码模板覆写成功");
      } else {
        // Execute Create
        const payload = { pathId: activePath.ID, content: codeContent };
        await createModel(payload);
        toast.success("全新代码模板落盘成功");
        // Refetch to sync ID
        const res: any = await getModelListByPathId(activePath.ID);
        let models = unwrapResponseData(res);
        if (!Array.isArray(models)) models = [];
        models = models.filter((x: any) => x.pathId === activePath.ID);
        if (models.length > 0) setActiveModel(models[0]);
      }
    } catch (err) {
      toast.error('网络操作中断');
    } finally {
      setSaving(false);
    }
  };

  // Determine language natively mapped to monaco using the fileName regex from the Go layer
  const getLanguageType = (filename: string) => {
    const fn = (filename || '').toLowerCase();
    if (fn.includes('.json')) return 'json';
    if (fn.includes('.sh') || fn.includes('.bash')) return 'shell';
    if (fn.includes('.py')) return 'python';
    if (fn.includes('.js') || fn.includes('.ts') || fn.includes('.vue')) return 'javascript';
    if (fn.includes('.java')) return 'java';
    if (fn.includes('.go')) return 'go';
    if (fn.includes('.xml') || fn.includes('.html')) return 'xml';
    if (fn.includes('.md')) return 'markdown';
    if (fn.includes('.yaml') || fn.includes('.yml')) return 'yaml';
    return 'plaintext';
  };

  // --- Tree Building Logic ---
  const filteredPaths = paths.filter(p => {
    if (!searchTerm) return true;
    const lowerQ = searchTerm.toLowerCase();
    return (p.fileName && p.fileName.toLowerCase().includes(lowerQ)) || (p.fileUrl && p.fileUrl.toLowerCase().includes(lowerQ));
  });

  const buildTree = (paths: any[]) => {
    const root: any = { name: 'root', isDir: true, children: {} };
    paths.forEach(p => {
      const parts = (p.fileUrl || '').split('/').filter(Boolean);
      let current = root;
      parts.forEach((part: string) => {
        if (!current.children[part]) {
          current.children[part] = { name: part, isDir: true, children: {} };
        }
        current = current.children[part];
      });
      if (p.fileName) {
        current.children[p.fileName] = { name: p.fileName, isDir: false, fileNode: p, children: {} };
      }
    });

    // Helper to convert object map to sorted array (directories first, then files)
    const toArray = (node: any): any[] => {
      const arr = Object.values(node.children);
      arr.sort((a: any, b: any) => {
        if (a.isDir === b.isDir) return a.name.localeCompare(b.name);
        return a.isDir ? -1 : 1;
      });
      arr.forEach((child: any) => {
        if (child.isDir) child.childrenArr = toArray(child);
      });
      return arr;
    };

    return toArray(root);
  };

  const treeData = buildTree(filteredPaths);

  // For managing expanded folders
  const [expandedFolders, setExpandedFolders] = useState<Record<string, boolean>>({});

  // Recursive Tree Render Component
  const TreeItem = ({ node, level, pathStr }: { node: any, level: number, pathStr: string }) => {
    const currentPath = `${pathStr}/${node.name}`;
    const isExpanded = expandedFolders[currentPath] !== false;

    const handleToggle = (e: React.MouseEvent) => {
      e.stopPropagation();
      if (node.isDir) {
        setExpandedFolders(prev => ({ ...prev, [currentPath]: prev[currentPath] === false ? true : false }));
      }
    };

    if (node.isDir) {
      return (
        <div className="w-full">
          <div 
            onClick={handleToggle}
            className="flex items-center gap-1.5 py-1.5 px-2 hover:bg-slate-200/50 rounded-lg cursor-pointer transition-colors text-slate-700"
            style={{ paddingLeft: `${level * 16 + 8}px` }}
          >
            <div className="w-4 flex items-center justify-center text-slate-400">
              {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            </div>
            <Folder size={14} className={isExpanded ? "text-teal-500 fill-teal-500/20" : "text-amber-500 fill-amber-500/20"} />
            <span className="text-sm font-medium truncate" title={node.name}>{node.name}</span>
          </div>
          {isExpanded && node.childrenArr && (
            <div className="w-full">
              {node.childrenArr.map((child: any) => (
                <TreeItem key={child.name} node={child} level={level + 1} pathStr={currentPath} />
              ))}
            </div>
          )}
        </div>
      );
    } else {
      // File node
      const p = node.fileNode;
      const isActive = activePath?.ID === p.ID;
      return (
        <div 
          onClick={() => handleSelectPath(p)}
          className={clsx(
            "flex items-center gap-2 py-1.5 px-2 my-0.5 rounded-lg cursor-pointer transition-colors text-sm w-full",
            isActive 
              ? "bg-teal-100 text-teal-900 shadow-sm border border-teal-200/50" 
              : "hover:bg-slate-200/50 text-slate-600 border border-transparent"
          )}
          style={{ paddingLeft: `${level * 16 + 8 + 16}px` }} // +16 for icon alignment without chevron
        >
          <FileCode size={14} className={clsx("flex-shrink-0", isActive ? "text-teal-600" : "text-slate-400")} />
          <span className="truncate flex-1" title={node.name}>{node.name}</span>
        </div>
      );
    }
  };

  return (
    <div className="flex flex-col w-full h-[calc(100vh-140px)] border border-slate-200/60 rounded-3xl overflow-hidden bg-white shadow-xl mt-2 animate-in fade-in duration-300">
      
      {/* Top Banner */}
      <div className="bg-gradient-to-r from-slate-800 to-slate-900 text-white px-6 py-4 flex items-center justify-between shadow-md z-20">
         <div className="flex items-center gap-4">
             <button onClick={() => navigate('/code-generate')} className="p-2 hover:bg-white/10 rounded-xl transition-colors flex items-center gap-1 text-sm font-medium border border-slate-700">
                 <ArrowLeft size={16} /> 返回主版
             </button>
             <div className="h-6 w-px bg-slate-700"></div>
             <div>
                <h1 className="text-lg font-bold tracking-tight text-teal-400">工程逻辑架构树与模型编辑 <span className="text-slate-300 font-normal">/ {projectName}</span></h1>
             </div>
         </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Left Sidebar: Scripts Explorer */}
        <div className="w-80 border-r border-slate-200 flex flex-col bg-[#f8fafc] flex-shrink-0 relative z-10">
          <div className="p-3 border-b border-slate-200 bg-white">
             <input type="text" value={searchTerm} onChange={e => setSearchTerm(e.target.value)} placeholder="过滤文件树..." className="w-full bg-slate-100/50 border border-slate-200 px-3 py-2 text-sm rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-500/30 transition-all font-mono" />
          </div>

          {/* File Tree */}
          <div className="flex-1 overflow-y-auto p-2 pb-6">
            {loadingList ? (
              <div className="text-sm text-slate-400 text-center mt-10">检索架构树...</div>
            ) : treeData.length === 0 ? (
              <div className="text-sm text-slate-400 text-center mt-10">
                {searchTerm ? '未找到节点' : '工程暂无路径'}
              </div>
            ) : (
              <div className="w-full flex flex-col">
                {treeData.map((child: any) => (
                  <TreeItem key={child.name} node={child} level={0} pathStr="" />
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right Content: Editor */}
        <div className="flex-1 flex flex-col bg-[#0f172a] min-w-0 relative overflow-hidden">
          {activePath ? (
            <>
              {/* Editor Header */}
              <div className="h-14 border-b border-slate-800 flex items-center justify-between px-6 bg-slate-900 select-none overflow-hidden shrink-0">
                    <div className="flex items-center gap-3">
                        <div className="px-2 py-1 bg-slate-800 rounded text-xs text-slate-400 font-mono tracking-widest uppercase shadow-inner border border-slate-700">TARGET</div>
                        <span className="text-slate-200 font-medium font-mono tracking-wide">{activePath.fileUrl}/{activePath.fileName}</span>
                    </div>
                  <button 
                    onClick={handleSaveContent}
                    disabled={saving || loadingContent}
                    className="flex-shrink-0 bg-teal-500 hover:bg-teal-400 disabled:opacity-50 text-slate-900 px-4 py-2 rounded-xl text-sm font-bold inline-flex items-center gap-2 transition-all whitespace-nowrap shadow-[0_0_15px_rgba(20,184,166,0.3)] hover:shadow-[0_0_25px_rgba(20,184,166,0.5)] active:scale-95"
                  >
                    {saving ? <RefreshCw size={16} className="animate-spin" /> : <Save size={16} />}
                    {activeModel?.ID ? '覆盖原落盘规则' : '初始化并强制落盘'}
                  </button>
              </div>
              
              {/* Monaco Canvas */}
              <div className="flex-1 relative">
                {loadingContent && (
                  <div className="absolute inset-0 z-10 flex items-center justify-center bg-slate-900/80 backdrop-blur-sm">
                    <span className="text-teal-500 animate-pulse font-bold tracking-widest text-sm bg-teal-500/10 px-4 py-2 rounded-lg border border-teal-500/30">解析模板缓冲区中...</span>
                  </div>
                )}
                <Editor
                    height="100%"
                    theme="vs-dark"
                    language={getLanguageType(activePath.fileName)}
                    value={codeContent}
                    onChange={(val) => setCodeContent(val || '')}
                    options={{
                      minimap: { enabled: false },
                      fontSize: 15,
                      fontFamily: '"Fira Code", Monaco, "Courier New", monospace',
                      wordWrap: "on",
                      padding: { top: 24, bottom: 24 },
                      cursorBlinking: "smooth",
                      smoothScrolling: true,
                      lineHeight: 1.6
                    }}
                />
              </div>
            </>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center text-slate-600 bg-slate-900">
              <div className="p-8 bg-slate-800/50 rounded-full mb-6">
                 <FileCode size={64} className="opacity-20 text-teal-500" />
              </div>
              <h2 className="text-2xl font-bold text-slate-400 mb-2">未定焦工作区</h2>
              <p className="text-slate-500 max-w-sm text-center">从左侧架构树图谱中选中任意代码生成节点，即刻进入无缝模板构建流程。</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
