import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Plus, Copy, FileCode, Search, Edit2, Trash2, Bot } from 'lucide-react';
import { getProjectList, createProject, updateProject, deleteProject, copyProject, globalReplace, generateCode, getGenerateRecordByUser } from '@/api/code_generate_project';
import { PlaceholderModal } from '@/components/PlaceholderModal';
import { useProjectStore } from '@/stores/useProjectStore';
import toast from 'react-hot-toast';

const getPublicGenerateCodeUrl = () => {
  const baseApi = import.meta.env.VITE_BASE_API || '/api';
  const normalizedBase = baseApi.replace(/\/$/, '');
  if (/^https?:\/\//i.test(normalizedBase)) {
    return `${normalizedBase}/tbgenerateproject/public/generateCode`;
  }
  const basePath = normalizedBase.startsWith('/') ? normalizedBase : `/${normalizedBase}`;
  return `${window.location.origin}${basePath}/tbgenerateproject/public/generateCode`;
};

const buildCodexInstruction = (project: any) => {
  const projectId = Number(project?.ID) || 0;
  const projectName = project?.projectName || '未命名项目';
  const diskPath = project?.diskPath || '未配置';
  const remark = project?.remark || '暂无说明';
  const endpoint = getPublicGenerateCodeUrl();

  return `你是一个没有历史记忆的 Codex。请按下面流程调用本地代码生成器，根据生成结果继续微调代码。

项目卡片信息:
- 项目名称: ${projectName}
- 项目 ID: ${projectId}
- 代码生成根目录: ${diskPath}
- 项目说明: ${remark}

生成器作用:
这个接口会根据 tableStructure 中的 CREATE TABLE SQL 解析表名、字段、注释和类型，再读取项目 ID=${projectId} 对应的路径模板、代码模板、项目占位符和全局占位符，最终把代码写入上面的代码生成根目录。卡片不同，模板和输出目录不同，生成结果也不同。

调用接口:
POST ${endpoint}

请求体:
\`\`\`json
{
  "projectId": ${projectId},
  "moduleName": "根据用户需求填写，例如 order",
  "moduleComment": "根据用户需求填写，例如 订单",
  "dbType": "mysql",
  "tableStructure": "用户提供的完整 CREATE TABLE SQL"
}
\`\`\`

执行规则:
1. 如果用户没有提供 moduleName、moduleComment、dbType 或 CREATE TABLE SQL，先向用户确认。
2. 使用 curl、fetch 或你可用的 HTTP 工具调用上面的 POST 接口。
3. 接口返回 code=0 才代表生成成功；data.diskPath 是代码根目录，data.generatedFiles 是本次生成或追加的文件。
4. 生成成功后不要停止。请打开 data.generatedFiles 中的文件；如果没有文件列表，就检查代码生成根目录。
5. 根据用户的业务输入、当前代码风格、编译错误和生成结果继续修改代码。
6. 相同表结构调用不同项目卡片，结果不同；同一项目卡片调用不同表结构，结果也不同。只有同一项目、同一表、同一参数和同一模板配置时，生成结果才应一致。
7. 不要把生成器项目本身当成目标项目，目标代码位置是: ${diskPath}
`;
};

export default function ProjectDashboard() {
  const navigate = useNavigate();
  const { activeProjectId } = useProjectStore();
  const [projects, setProjects] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [showReplaceModal, setShowReplaceModal] = useState(false);
  const [showPlaceholderModal, setShowPlaceholderModal] = useState(false);
  const [showCodexModal, setShowCodexModal] = useState(false);
  const [currentProject, setCurrentProject] = useState<any>({});
  const [codexProject, setCodexProject] = useState<any>(null);
  
  const defaultGenForm = { id: '', moduleName: '', moduleComment: '', tableStructure: '', dbType: 'mysql' };
  const [genForm, setGenForm] = useState(defaultGenForm);
  const [replaceForm, setReplaceForm] = useState({ id: 0, formerStr: '', replaceStr: '' });

  const fetchProjects = async () => {
    setLoading(true);
    try {
      const res = await getProjectList({ projectConfigId: activeProjectId });
      let data = res.data;
      if (data && !Array.isArray(data) && Array.isArray(data.list)) {
        data = data.list;
      }
      setProjects(Array.isArray(data) ? data : []);
    } catch (e) {
      toast.error('获取项目列表失败');
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchProjects();
  }, [activeProjectId]);

  const handleSave = async () => {
    try {
      if (currentProject.ID) {
        await updateProject(currentProject);
        toast.success('更新成功');
      } else {
        await createProject({ ...currentProject, userName: 'conchi', projectConfigId: activeProjectId || 0 });
        toast.success('创建成功');
      }
      setShowModal(false);
      fetchProjects();
    } catch (e) {
      toast.error('保存失败');
    }
  };

  const handleDelete = async (data: any) => {
    if (confirm('确定删除该项目及其所有配置吗？')) {
      try {
        await deleteProject(data);
        toast.success('删除成功');
        fetchProjects();
      } catch (e) {
        toast.error('删除失败');
      }
    }
  };

  const handleCopy = async (id: string) => {
    try {
      await copyProject(id);
      toast.success('复制成功');
      fetchProjects();
    } catch (e) {
      toast.error('复制失败');
    }
  };

  const openGenerateModal = async (projectId: string | number) => {
    const id = projectId.toString();
    setGenForm({ ...defaultGenForm, id });
    setShowGenerateModal(true);
    try {
      const res: any = await getGenerateRecordByUser(id);
      const record = res?.data;
      if (record) {
        setGenForm({
          id,
          moduleName: record.moduleName || '',
          moduleComment: record.moduleComment || '',
          tableStructure: record.tableStructure || '',
          dbType: record.dbType || 'mysql',
        });
      }
    } catch (e) {
      toast.error('读取上次生成参数失败');
    }
  };



  const handleGenerate = async () => {
    if (!genForm.tableStructure) { toast.error('建表SQL不能为空'); return; }
    try {
      await generateCode(genForm);
      toast.success('代码成功生成完毕并落盘');
      setShowGenerateModal(false);
    } catch (e) {
      toast.error('代码生成系统异常');
    }
  };

  const handleReplace = async () => {
    if (!replaceForm.formerStr) { toast.error('原字符必填'); return; }
    try {
      await globalReplace(replaceForm);
      toast.success('工程全局替换结束');
      setShowReplaceModal(false);
    } catch(e) {
      toast.error('替换中断');
    }
  };

  const openCodexModal = (project: any) => {
    setCodexProject(project);
    setShowCodexModal(true);
  };

  const copyCodexInstruction = async () => {
    if (!codexProject) return;
    try {
      await navigator.clipboard.writeText(buildCodexInstruction(codexProject));
      toast.success('Codex 指令已复制');
    } catch (e) {
      toast.error('复制失败，请手动复制');
    }
  };

  const codexInstruction = codexProject ? buildCodexInstruction(codexProject) : '';

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-8 animate-fade-in">
      {/* Header Area */}
      <div className="flex justify-between items-center bg-white/50 backdrop-blur-xl p-6 rounded-3xl shadow-sm border border-slate-200/60 transition-colors">
        <div>
          <h1 className="text-3xl font-extrabold text-slate-800 tracking-tight">项目代码空间</h1>
          <p className="text-slate-500 mt-1">管理、配置您的所有自动化代码库工程模板</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => setShowPlaceholderModal(true)}
            className="flex items-center gap-2 bg-white hover:bg-slate-50 text-slate-700 px-5 py-3 rounded-xl shadow-sm border border-slate-200 transition-all hover:scale-105 active:scale-95"
          >
            <FileCode size={20} className="text-violet-500" />
            <span>全局占位符管理</span>
          </button>
          <button
            onClick={() => {
              setCurrentProject({});
              setShowModal(true);
            }}
            className="flex items-center gap-2 bg-gradient-to-r from-teal-500 to-emerald-500 hover:from-teal-600 hover:to-emerald-600 text-white px-5 py-3 rounded-xl shadow-lg shadow-teal-500/20 transition-all hover:scale-105 active:scale-95"
          >
            <Plus size={20} />
            <span>新建项目</span>
          </button>
        </div>
      </div>

      {/* Grid List */}
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map(i => (
            <div key={i} className="h-48 bg-slate-100 rounded-3xl animate-pulse"></div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <AnimatePresence>
            {projects.map((p: any) => (
              <motion.div
                layout
                initial={{ opacity: 0, y: 30 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.9 }}
                key={p.ID}
                className="group relative min-w-0 bg-white rounded-3xl p-6 shadow-sm hover:shadow-2xl hover:shadow-teal-500/10 border border-slate-100 transition-all duration-300 overflow-hidden"
              >
                {/* Decorative blob */}
                <div className="absolute -top-10 -right-10 w-32 h-32 bg-teal-500/5 rounded-full blur-2xl group-hover:bg-teal-500/10 transition-colors"></div>

                <div className="flex justify-between items-start gap-3 mb-6 relative z-10">
                  <div className="flex min-w-0 items-center gap-3">
                    <div className="p-3 bg-teal-50 text-teal-600 rounded-2xl">
                      <FileCode size={24} />
                    </div>
                    <div className="min-w-0">
                      <h3 className="font-bold text-lg text-slate-800 truncate" title={p.projectName}>{p.projectName}</h3>
                      <p className="text-xs text-slate-400 font-mono">ID: {p.ID}</p>
                    </div>
                  </div>
                  <div className="flex shrink-0 gap-1">
                    <button onClick={() => { setCurrentProject(p); setShowModal(true); }} className="p-2 text-slate-400 hover:text-teal-600 hover:bg-teal-50 rounded-lg transition-colors"><Edit2 size={16} /></button>
                    <button onClick={() => handleDelete(p)} className="p-2 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"><Trash2 size={16} /></button>
                  </div>
                </div>

                <div className="space-y-3 relative z-10">
                  <div className="min-w-0 overflow-hidden text-sm font-medium text-slate-600 bg-slate-50 p-3 rounded-xl border border-slate-100/50">
                    <span className="block text-xs text-slate-400 mb-1">磁盘输出路径</span>
                    <span className="block min-w-0 break-all leading-relaxed" title={p.diskPath || '未配置'}>
                      {p.diskPath || '未配置'}
                    </span>
                  </div>
                  <div className="text-sm text-slate-500 line-clamp-2">
                    {p.remark || '暂无备注说明...'}
                  </div>
                </div>

                <div className="mt-6 pt-4 border-t border-slate-100 flex flex-col gap-2 relative z-10">
                  <div className="flex gap-2">
                    <button onClick={() => navigate(`/code-generate/${p.ID}/templates`)} className="flex-1 flex justify-center items-center gap-2 py-2.5 text-sm text-indigo-700 bg-indigo-50 hover:bg-indigo-100 rounded-xl transition-colors font-bold border border-indigo-200">
                      <FileCode size={16} /> 编辑引擎
                    </button>
                    <button onClick={() => navigate(`/code-generate/${p.ID}/placeholders`)} className="flex-1 flex justify-center items-center gap-2 py-2.5 text-sm text-amber-700 bg-amber-50 hover:bg-amber-100 rounded-xl transition-colors font-bold border border-amber-200">
                      <FileCode size={16} /> 占位符池
                    </button>
                  </div>
                  <div className="flex gap-2">
                    <button onClick={() => openGenerateModal(p.ID)} className="flex-1 flex justify-center items-center gap-2 py-2 text-sm text-white bg-teal-600 hover:bg-teal-700 rounded-xl transition-colors font-medium">
                      <FileCode size={16} /> 引擎生成代码
                    </button>
                    <button onClick={() => { setReplaceForm({ ...replaceForm, id: p.ID }); setShowReplaceModal(true); }} className="flex-1 flex justify-center items-center gap-2 py-2 text-sm text-white bg-indigo-600 hover:bg-indigo-700 rounded-xl transition-colors font-medium">
                      <Search size={16} /> 全局替换
                    </button>
                  </div>
                  <div className="flex gap-2">
                    <button onClick={() => openCodexModal(p)} className="flex-1 flex justify-center items-center gap-2 py-2 text-sm text-slate-600 bg-white hover:bg-slate-50 border border-slate-200 rounded-xl transition-colors">
                      <Bot size={16} /> Codex 指令
                    </button>
                    <button onClick={() => handleCopy(p.ID)} className="flex-1 flex justify-center items-center gap-2 py-2 text-sm text-slate-600 bg-white hover:bg-slate-50 border border-slate-200 rounded-xl transition-colors">
                      <Copy size={16} /> 克隆
                    </button>
                  </div>
                </div>
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      )}

      {/* Modal */}
      <AnimatePresence>
        {showModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <motion.div
              initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm"
              onClick={() => setShowModal(false)}
            />
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }}
              className="relative w-full max-w-lg bg-white rounded-3xl shadow-2xl p-8"
            >
              <h2 className="text-2xl font-bold text-slate-800 mb-6">{currentProject.ID ? '编辑项目' : '新建项目'}</h2>
              <div className="space-y-5">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">项目名称</label>
                  <input
                    type="text"
                    value={currentProject.projectName || ''}
                    onChange={e => setCurrentProject({ ...currentProject, projectName: e.target.value })}
                    className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-teal-500/50 transition-all"
                    placeholder="输入项目名, 比如 Easy Backend..."
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">落盘路径</label>
                  <input
                    type="text"
                    value={currentProject.diskPath || ''}
                    onChange={e => setCurrentProject({ ...currentProject, diskPath: e.target.value })}
                    className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-teal-500/50 transition-all"
                    placeholder="输入代码生成的根目录路径"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">项目描述</label>
                  <textarea
                    value={currentProject.remark || ''}
                    onChange={e => setCurrentProject({ ...currentProject, remark: e.target.value })}
                    className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-teal-500/50 transition-all min-h-[100px]"
                    placeholder="简单的描述记录..."
                  />
                </div>
              </div>
              <div className="mt-8 flex justify-end gap-3">
                <button
                  onClick={() => setShowModal(false)}
                  className="px-5 py-2.5 text-slate-600 hover:bg-slate-100 rounded-xl transition-colors font-medium"
                >
                  取消
                </button>
                <button
                  onClick={handleSave}
                  className="px-5 py-2.5 bg-slate-800 hover:bg-slate-900 text-white rounded-xl transition-colors font-medium shadow-lg shadow-slate-900/20"
                >
                  保存设置
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      <PlaceholderModal isOpen={showPlaceholderModal} onClose={() => setShowPlaceholderModal(false)} />

      {/* Codex Instruction Modal */}
      <AnimatePresence>
        {showCodexModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm" onClick={() => setShowCodexModal(false)} />
            <motion.div initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }} className="relative w-full max-w-3xl bg-white rounded-3xl shadow-2xl p-8 max-h-[90vh] overflow-y-auto">
              <h2 className="text-2xl font-bold text-slate-800 mb-6 flex items-center gap-2"><Bot className="text-teal-500" /> Codex 自动生成指令</h2>
              <textarea
                readOnly
                value={codexInstruction}
                className="w-full min-h-[520px] px-4 py-3 bg-slate-950 text-slate-100 font-mono text-sm border border-slate-800 rounded-2xl focus:outline-none focus:ring-2 focus:ring-teal-500/50 leading-relaxed"
              />
              <div className="mt-6 flex justify-end gap-3">
                <button onClick={() => setShowCodexModal(false)} className="px-5 py-2.5 text-slate-600 hover:bg-slate-100 rounded-xl">关闭窗口</button>
                <button onClick={copyCodexInstruction} className="px-5 py-2.5 bg-teal-600 hover:bg-teal-700 text-white rounded-xl shadow-lg shadow-teal-600/20 flex items-center gap-2"><Copy size={16} /> 复制指令</button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      {/* Global Replace Modal */}
      <AnimatePresence>
        {showReplaceModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm" onClick={() => setShowReplaceModal(false)} />
            <motion.div initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }} className="relative w-full max-w-lg bg-white rounded-3xl shadow-2xl p-8">
              <h2 className="text-2xl font-bold text-slate-800 mb-6">全工程全局字符串替换</h2>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">原字符串 (formerStr)</label>
                  <input type="text" value={replaceForm.formerStr} onChange={e => setReplaceForm({ ...replaceForm, formerStr: e.target.value })} className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/50" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">替换为什么字符 (replaceStr)</label>
                  <input type="text" value={replaceForm.replaceStr} onChange={e => setReplaceForm({ ...replaceForm, replaceStr: e.target.value })} className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/50" />
                </div>
              </div>
              <div className="mt-8 flex justify-end gap-3">
                <button onClick={() => setShowReplaceModal(false)} className="px-5 py-2.5 text-slate-600 hover:bg-slate-100 rounded-xl">取消</button>
                <button onClick={handleReplace} className="px-5 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl">执行全盘替换</button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      {/* Code Generate Modal */}
      <AnimatePresence>
        {showGenerateModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm" onClick={() => setShowGenerateModal(false)} />
            <motion.div initial={{ opacity: 0, scale: 0.95, y: 20 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 20 }} className="relative w-full max-w-xl bg-white rounded-3xl shadow-2xl p-8 max-h-[90vh] overflow-y-auto">
              <h2 className="text-2xl font-bold text-slate-800 mb-6 flex items-center gap-2"><FileCode className="text-teal-500" /> 代码生成引擎与落盘器</h2>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">解析数据库方言 (DB Type)</label>
                  <select value={genForm.dbType} onChange={e => setGenForm({ ...genForm, dbType: e.target.value })} className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-teal-500/50">
                    <option value="mysql">MySQL</option>
                    <option value="postgresql">PostgreSQL</option>
                    <option value="mssql">SQL Server</option>
                    <option value="oracle">Oracle</option>
                    <option value="sqlite">SQLite</option>
                  </select>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-2">引擎识别模块名 (moduleName)</label>
                    <input type="text" value={genForm.moduleName} onChange={e => setGenForm({ ...genForm, moduleName: e.target.value })} className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-teal-500/50" placeholder="例如: user" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-2">模块注释 (moduleComment)</label>
                    <input type="text" value={genForm.moduleComment} onChange={e => setGenForm({ ...genForm, moduleComment: e.target.value })} className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-teal-500/50" placeholder="例如: 用户表" />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-2">CREATE TABLE 完整建表语句 (用于逆向推理)</label>
                  <textarea value={genForm.tableStructure} onChange={e => setGenForm({ ...genForm, tableStructure: e.target.value })} className="w-full px-4 py-3 bg-slate-50 font-mono text-sm border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-teal-500/50 min-h-[250px]" placeholder="CREATE TABLE `tb_domain` ..." />
                </div>
              </div>
              <div className="mt-8 flex justify-end gap-3">
                <button onClick={() => setShowGenerateModal(false)} className="px-5 py-2.5 text-slate-600 hover:bg-slate-100 rounded-xl">关闭窗口</button>
                <button onClick={handleGenerate} className="px-5 py-2.5 bg-teal-600 hover:bg-teal-700 text-white rounded-xl shadow-lg shadow-teal-600/20">开始一键编译与落盘提取</button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </div>
  );
};
