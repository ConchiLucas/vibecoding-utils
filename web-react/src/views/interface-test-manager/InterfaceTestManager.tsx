import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { Play, Code2, User, Globe2 } from 'lucide-react';
import { getTbInterfaceEnvList, TbInterfaceEnv } from '../../api/sysInterfaceEnv';
import { getTbServerUserList, TbServerUser } from '../../api/sysInterfaceServerUser';
import { getParamsEntity } from '../../api/sysInterfaceParams';
import { forwardInterfaceApi } from '../../api/sysInterface';

interface InterfaceTestManagerProps {
  interfaceId: number;
  interfacePaths: string;
  projectName?: string;
}

export default function InterfaceTestManager({ interfaceId, interfacePaths, projectName }: InterfaceTestManagerProps) {
  const [loading, setLoading] = useState(false);
  const [environments, setEnvironments] = useState<TbInterfaceEnv[]>([]);
  const [users, setUsers] = useState<TbServerUser[]>([]);

  // Form State
  const [environment, setEnvironment] = useState('');
  const [selectedUserId, setSelectedUserId] = useState('');
  const [requestParam, setRequestParam] = useState('');
  const [responseParam, setResponseParam] = useState('');
  const [paramsId, setParamsId] = useState<number | undefined>();
  // identity from last-used params — used to auto-select the user after users load
  const [pendingIdentity, setPendingIdentity] = useState<string>('');

  const formatToJson = (jsonStr: string) => {
    try {
      if (!jsonStr.trim()) return jsonStr;
      const parsed = JSON.parse(jsonStr);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return jsonStr; // invalid JSON, do not format
    }
  };

  // 1. Load Environments
  useEffect(() => {
    if (!projectName) return;
    setEnvironment('');
    setSelectedUserId('');
    setRequestParam('');
    setResponseParam('');
    setParamsId(undefined);
    setPendingIdentity('');

    getTbInterfaceEnvList({ page: 1, pageSize: 999, projectName })
      .then(res => setEnvironments(res.data?.list || []))
      .catch(err => console.error("Failed to load environments:", err));

    // Load initial params via getParamsEntity (mirrors Python params/entity logic)
    if (interfaceId) {
      getParamsEntity({ id: interfaceId, paths: interfacePaths })
        .then(res => {
          if (res.data) {
            const entity = res.data;
            setEnvironment(entity.environment || '');
            setRequestParam(formatToJson(entity.interfaceParams || ''));
            setResponseParam(formatToJson(entity.responseParams || ''));
            setParamsId(entity.paramsId || undefined);
            setPendingIdentity(entity.identity || '');
          }
        })
        .catch(err => console.error('Failed to load interface params:', err));
    }
  }, [interfaceId, interfacePaths, projectName]);

  // 2. Load Users when environment changes
  useEffect(() => {
    if (!environment || !projectName) {
      setUsers([]);
      setSelectedUserId('');
      return;
    }
    getTbServerUserList({ page: 1, pageSize: 999, projectName, environment })
      .then(res => {
         const list = res.data?.list || [];
         setUsers(list);
         // Try to restore last-used identity; fall back to first user
         if (pendingIdentity) {
           const matched = list.find((u: TbServerUser) => {
             const candidates = [
               String(u.ID),
               u.roleCode,
               u.loginAccount,
               u.userNickname,
               getUserDisplayName(u),
             ].filter(Boolean);
             return candidates.includes(pendingIdentity);
           });
           setSelectedUserId(matched ? String(matched.ID) : (list.length > 0 ? String(list[0].ID) : ''));
         } else if (list.length > 0) {
           setSelectedUserId(String(list[0].ID));
         } else {
           setSelectedUserId('');
         }
      })
      .catch(err => console.error('Failed to load users:', err));
  }, [environment, projectName, pendingIdentity]);

  const handleFormatRequest = () => {
    setRequestParam(prev => formatToJson(prev));
  };

  const getUserDisplayName = (user: TbServerUser) => {
    if (user.roleName) {
      return `${user.userNickname || user.loginAccount}/${user.roleName}`;
    }
    return user.userNickname || user.loginAccount;
  };

  const handleSend = async () => {
    if (!environment) {
      toast.error('请选择测试环境');
      return;
    }
    setLoading(true);
    setResponseParam(''); // clear previous

    const selectedEnv = environments.find(e => e.envName === environment);
    // Find selected user's header
    const selectedUser = users.find(u => String(u.ID) === selectedUserId);

    try {
      const res = await forwardInterfaceApi({
         id: interfaceId,
         paramsId: paramsId,
         environment: selectedEnv?.envName || environment,
         requestParam: requestParam,
         clientId: selectedUserId ? Number(selectedUserId) : undefined,
         requestHeader: selectedUser?.requestHeader || undefined
      });
      
      if (res.code === 0 || res.code === 200) {
         toast.success(res.msg || '请求执行成功');
         try {
            setResponseParam(JSON.stringify(res.data, null, 2));
         } catch {
            setResponseParam(JSON.stringify(res, null, 2));
         }
         getParamsEntity({ id: interfaceId, paths: interfacePaths })
           .then(entityRes => {
             if (entityRes.data?.paramsId) {
               setParamsId(entityRes.data.paramsId);
             }
           })
           .catch(err => console.error('Failed to refresh interface params:', err));
      } else {
         toast.error(res.msg || '执行失败');
         setResponseParam(JSON.stringify(res, null, 2));
      }
    } catch (error: any) {
      console.error(error);
      toast.error('请求后端接口服务出错');
      setResponseParam(error?.message || '请求失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-full space-y-6">
       
       <div className="flex items-center gap-4 border-b border-slate-200 pb-4">
           <div className="flex-1 grid grid-cols-2 gap-6">
              {/* Environment Select */}
              <div>
                 <label className="flex items-center gap-2 text-sm font-semibold text-slate-700 mb-2">
                    <Globe2 size={16} /> 运行环境
                 </label>
                 <select 
                    value={environment}
                    onChange={e => setEnvironment(e.target.value)}
                    className="w-full px-4 py-2 border border-slate-300 rounded-xl bg-slate-50 focus:bg-white focus:ring-2 focus:ring-indigo-500 transition outline-none"
                 >
                    <option value="">请选择运行环境</option>
                    {environments.map(env => (
                       <option key={env.ID} value={env.envName}>{env.envName} ({env.baseUrl})</option>
                    ))}
                 </select>
              </div>

              {/* User Select */}
              <div>
                 <label className="flex items-center gap-2 text-sm font-semibold text-slate-700 mb-2">
                    <User size={16} /> 选择用户
                 </label>
                 <select 
                    value={selectedUserId}
                    onChange={e => setSelectedUserId(e.target.value)}
                    disabled={users.length === 0}
                    className="w-full px-4 py-2 border border-slate-300 rounded-xl bg-slate-50 focus:bg-white focus:ring-2 focus:ring-indigo-500 transition outline-none disabled:opacity-50 disabled:cursor-not-allowed"
                 >
                    {users.length === 0 ? (
                       <option value="">暂无可用用户</option>
                    ) : (
                       <>
                         <option value="">请选择用户</option>
                         {users.map(user => (
                            <option key={user.ID} value={String(user.ID)}>
                               {getUserDisplayName(user)}
                            </option>
                         ))}
                       </>
                    )}
                 </select>
              </div>
           </div>
           
           <div className="shrink-0 flex items-end ml-4 self-end space-y-2">
              <button 
                onClick={handleSend}
                disabled={loading}
                className="flex items-center justify-center gap-2 px-8 py-3 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold rounded-xl transition shadow-md hover:shadow-lg disabled:opacity-50 disabled:cursor-not-allowed"
              >
                 {loading ? (
                    <span className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
                 ) : (
                    <Play size={18} fill="currentColor" />
                 )}
                 执行测试
              </button>
           </div>
       </div>

       <div className="flex-1 grid grid-cols-1 lg:grid-cols-2 gap-6 min-h-0">
          
          {/* Request Params */}
          <div className="flex flex-col bg-white border border-slate-200 rounded-2xl shadow-sm min-h-0">
             <div className="shrink-0 px-4 py-3 flex items-center justify-between border-b border-slate-100 bg-slate-50/50 rounded-t-2xl">
                 <span className="text-sm font-semibold text-slate-700 flex items-center gap-2">
                    <Code2 size={16} className="text-indigo-500"/> JSON 请求参数
                 </span>
                 <button onClick={handleFormatRequest} className="text-xs text-indigo-600 hover:text-indigo-800 font-medium px-2 py-1 hover:bg-indigo-50 rounded transition">格式化 (Format)</button>
             </div>
             <textarea 
                value={requestParam}
                onChange={e => setRequestParam(e.target.value)}
                onBlur={handleFormatRequest}
                className="flex-1 w-full p-4 resize-none outline-none font-mono text-sm text-slate-700 bg-transparent"
                spellCheck={false}
                placeholder="请输入请求体 JSON 字符串..."
             />
          </div>

          {/* Response Params */}
          <div className="flex flex-col bg-slate-950 rounded-2xl shadow-sm min-h-0 relative overflow-hidden">
             <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(56,189,248,0.05),transparent_50%)] pointer-events-none"></div>
             <div className="shrink-0 px-4 py-3 flex items-center justify-between border-b border-white/10 bg-white/5 relative z-10">
                 <span className="text-sm font-semibold text-slate-200 flex items-center gap-2">
                    JSON 返回参数
                 </span>
             </div>
             <textarea 
                value={responseParam}
                readOnly
                className="flex-1 w-full p-4 resize-none outline-none font-mono text-sm text-emerald-400 bg-transparent relative z-10"
                spellCheck={false}
                placeholder="等待执行测试获取响应..."
             />
          </div>

       </div>

    </div>
  );
}
