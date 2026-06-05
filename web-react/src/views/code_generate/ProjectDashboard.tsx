import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Plus, Copy, ClipboardCopy, Database, FileCode, Edit2, Trash2 } from 'lucide-react';
import { getProjectList, createProject, updateProject, deleteProject, copyProject } from '@/api/code_generate_project';
import { getDbTemplateScripts, getDbTemplateTypes } from '@/api/db_template';
import { getModelList, getPathList } from '@/api/path_model';
import { useProjectStore } from '@/stores/useProjectStore';
import toast from 'react-hot-toast';

const unwrapResponseData = (res: any) => {
  return res?.data?.data ?? res?.data ?? [];
};

const normalizeProjectRows = (value: any) => {
  if (Array.isArray(value)) return value;
  if (value && Array.isArray(value.list)) return value.list;
  return [];
};

const copyTextToClipboard = async (text: string) => {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }

  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  textarea.style.top = '-9999px';
  textarea.setAttribute('readonly', 'readonly');
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const copied = document.execCommand('copy');
  document.body.removeChild(textarea);
  if (!copied) {
    throw new Error('copy failed');
  }
};

const languageForFilename = (filename: string) => {
  const lower = String(filename || '').toLowerCase();
  if (lower.endsWith('.java')) return 'java';
  if (lower.endsWith('.sql')) return 'sql';
  if (lower.endsWith('.xml')) return 'xml';
  if (lower.endsWith('.json')) return 'json';
  if (lower.endsWith('.yaml') || lower.endsWith('.yml')) return 'yaml';
  if (lower.endsWith('.sh')) return 'bash';
  if (lower.endsWith('.ts') || lower.endsWith('.tsx')) return 'tsx';
  if (lower.endsWith('.js') || lower.endsWith('.jsx')) return 'jsx';
  return 'text';
};

const pathLabel = (pathObj: any) => {
  const dir = String(pathObj.fileUrl || '').replace(/\/+/g, '/');
  const fileName = String(pathObj.fileName || '');
  if (!dir) return fileName || '未命名文件';
  return `${dir}${dir.endsWith('/') ? '' : '/'}${fileName}`.replace(/\/+/g, '/');
};

const isTableStructureNode = (pathObj: any, content: string) => {
  const fileName = String(pathObj.fileName || '').toLowerCase();
  return fileName.startsWith('create_') || /\bcreate\s+table\b/i.test(content);
};

const formatCodeSection = (title: string, pathObj: any, content: string) => [
  `### ${title}`,
  `路径：${pathLabel(pathObj)}`,
  '',
  `\`\`\`${languageForFilename(pathObj.fileName || '')}`,
  content.trim(),
  '```',
].join('\n');

const buildDbTemplateSqlSections = async (project: any, options: { markdown?: boolean } = {}) => {
  const projectId = Number(project.ID || 0);
  const typeRes: any = await getDbTemplateTypes(projectId);
  const types = Array.isArray(unwrapResponseData(typeRes)) ? unwrapResponseData(typeRes) : [];
  const sections: string[] = [];

  for (const typeObj of types) {
    const scriptRes: any = await getDbTemplateScripts(projectId, Number(typeObj.ID));
    const scripts = Array.isArray(unwrapResponseData(scriptRes)) ? unwrapResponseData(scriptRes) : [];

    scripts
      .filter((script: any) => String(script.content || '').trim())
      .forEach((script: any) => {
        if (options.markdown) {
          sections.push([
            `### 数据库模板 SQL`,
            `项目：${project.projectName || projectId}`,
            `业务类型：${typeObj.typeName || '-'}`,
            `脚本：${script.scriptName || typeObj.typeName || '-'}`,
            '',
            '```sql',
            String(script.content || '').trim(),
            '```',
          ].join('\n'));
          return;
        }

        sections.push([
          `-- 项目：${project.projectName || projectId}`,
          `-- 业务类型：${typeObj.typeName || '-'}`,
          `-- 脚本：${script.scriptName || typeObj.typeName || '-'}`,
          '',
          String(script.content || '').trim(),
        ].join('\n'));
      });
  }

  return sections;
};

export default function ProjectDashboard() {
  const navigate = useNavigate();
  const { activeProjectId } = useProjectStore();
  const [projects, setProjects] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [currentProject, setCurrentProject] = useState<any>({});
  const [copyingTemplateProjectId, setCopyingTemplateProjectId] = useState<number | null>(null);
  const [copyingCodeProjectId, setCopyingCodeProjectId] = useState<number | null>(null);

  const fetchProjects = async () => {
    setLoading(true);
    try {
      const res = await getProjectList({ projectConfigId: activeProjectId });
      setProjects(normalizeProjectRows(unwrapResponseData(res)));
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

  const handleCopyDbTemplateSql = async (project: any) => {
    const projectId = Number(project.ID || 0);
    if (!projectId) return;

    setCopyingTemplateProjectId(projectId);
    try {
      const sections = await buildDbTemplateSqlSections(project);

      if (sections.length === 0) {
        toast.error('该项目暂无可复制的 SQL 内容');
        return;
      }

      await copyTextToClipboard(sections.join('\n\n\n'));
      toast.success('数据库模板 SQL 已复制');
    } catch (e) {
      toast.error('复制数据库模板失败');
    } finally {
      setCopyingTemplateProjectId(null);
    }
  };

  const handleCopyProjectCodeForAi = async (project: any) => {
    const projectId = Number(project.ID || 0);
    if (!projectId) return;

    setCopyingCodeProjectId(projectId);
    try {
      const pathRes: any = await getPathList(projectId);
      let paths = normalizeProjectRows(unwrapResponseData(pathRes));
      paths = paths
        .filter((pathObj: any) => Number(pathObj.projectId || 0) === projectId)
        .sort((a: any, b: any) => pathLabel(a).localeCompare(pathLabel(b)));

      if (paths.length === 0) {
        toast.error('该项目暂无代码文件');
        return;
      }

      const modelRes: any = await getModelList();
      const models = normalizeProjectRows(unwrapResponseData(modelRes));
      const modelByPathId = new Map<number, any>();
      models.forEach((model: any) => {
        const pathId = Number(model.pathId || 0);
        if (pathId && !modelByPathId.has(pathId)) {
          modelByPathId.set(pathId, model);
        }
      });

      const tableSections: string[] = [];
      const codeSections: string[] = [];
      const dbTemplateSqlSections = await buildDbTemplateSqlSections(project, { markdown: true });

      paths.forEach((pathObj: any) => {
        const model = modelByPathId.get(Number(pathObj.ID || 0));
        const content = String(model?.content || '').trim();
        if (!content) return;

        if (isTableStructureNode(pathObj, content)) {
          tableSections.push(formatCodeSection('表结构 SQL', pathObj, content));
          return;
        }
        codeSections.push(formatCodeSection('代码文件', pathObj, content));
      });

      if (dbTemplateSqlSections.length === 0 && tableSections.length === 0 && codeSections.length === 0) {
        toast.error('该项目暂无可复制的代码或表结构内容');
        return;
      }

      const payload = [
        '# 请基于以下现有代码和表结构生成同风格代码',
        '',
        `项目：${project.projectName || projectId}`,
        `项目 ID：${projectId}`,
        `磁盘输出路径：${project.diskPath || '未配置'}`,
        '',
        '要求：请参考这些真实文件的包路径、Controller/Service/Dao/Domain/Model/SQL 写法，为新表生成同风格后端代码。',
        '',
        dbTemplateSqlSections.length ? '## 新表结构 SQL / 数据库模板 SQL' : '',
        dbTemplateSqlSections.join('\n\n'),
        '',
        tableSections.length ? '## 编辑引擎中的表结构 SQL' : '',
        tableSections.join('\n\n'),
        '',
        codeSections.length ? '## 完整代码文件' : '',
        codeSections.join('\n\n'),
      ].filter((section) => String(section).trim()).join('\n\n');

      await copyTextToClipboard(payload);
      toast.success('代码和表结构已复制');
    } catch (e) {
      toast.error('复制代码失败');
    } finally {
      setCopyingCodeProjectId(null);
    }
  };

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
                    <button onClick={() => navigate(`/code-generate/${p.ID}/db-templates`)} className="flex-1 flex justify-center items-center gap-2 py-2.5 text-sm text-cyan-800 bg-cyan-50 hover:bg-cyan-100 rounded-xl transition-colors font-bold border border-cyan-200">
                      <Database size={16} /> 数据库模板
                    </button>
                    <button onClick={() => handleCopyDbTemplateSql(p)} disabled={copyingTemplateProjectId === Number(p.ID)} className="flex-1 flex justify-center items-center gap-2 py-2.5 text-sm text-emerald-700 bg-emerald-50 hover:bg-emerald-100 rounded-xl transition-colors font-bold border border-emerald-200 disabled:cursor-not-allowed disabled:opacity-60">
                      <ClipboardCopy size={16} /> {copyingTemplateProjectId === Number(p.ID) ? '复制中' : '复制 SQL'}
                    </button>
                  </div>
                  <div className="flex gap-2">
                    <button onClick={() => navigate(`/code-generate/${p.ID}/templates`)} className="flex-1 min-w-0 flex justify-center items-center gap-2 py-2.5 px-2 text-xs text-indigo-700 bg-indigo-50 hover:bg-indigo-100 rounded-xl transition-colors font-bold border border-indigo-200 sm:text-sm">
                      <FileCode size={16} /> 编辑引擎
                    </button>
                    <button onClick={() => handleCopyProjectCodeForAi(p)} disabled={copyingCodeProjectId === Number(p.ID)} className="flex-1 min-w-0 flex justify-center items-center gap-2 py-2.5 px-2 text-xs text-amber-700 bg-amber-50 hover:bg-amber-100 border border-amber-200 rounded-xl transition-colors font-bold disabled:cursor-not-allowed disabled:opacity-60 sm:text-sm">
                      <ClipboardCopy size={16} /> {copyingCodeProjectId === Number(p.ID) ? '复制中' : '复制代码'}
                    </button>
                    <button onClick={() => handleCopy(p.ID)} className="flex-1 min-w-0 flex justify-center items-center gap-2 py-2.5 px-2 text-xs text-slate-600 bg-white hover:bg-slate-50 border border-slate-200 rounded-xl transition-colors sm:text-sm">
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

    </div>
  );
};
