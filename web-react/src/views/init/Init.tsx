import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import toast from 'react-hot-toast';
import { initDB } from '../../api/initdb';
import { Database, Server, Key, User, ShieldCheck } from 'lucide-react';

export default function Init() {
  const navigate = useNavigate();
  const [isLoading, setIsLoading] = useState(false);

  const [form, setForm] = useState({
    adminPassword: '123456',
    dbType: 'mysql',
    host: '127.0.0.1',
    port: '3306',
    userName: 'root',
    password: '',
    dbName: 'gva',
    dbPath: '',
    template: 'template0'
  });

  const handleDBTypeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const val = e.target.value;
    switch (val) {
      case 'mysql':
        setForm(prev => ({ ...prev, dbType: val, host: '127.0.0.1', port: '3306', userName: 'root', dbName: 'gva' }));
        break;
      case 'pgsql':
        setForm(prev => ({ ...prev, dbType: val, host: '127.0.0.1', port: '5432', userName: 'postgres', dbName: 'gva' }));
        break;
      case 'oracle':
        setForm(prev => ({ ...prev, dbType: val, host: '127.0.0.1', port: '1521', userName: 'oracle', dbName: 'gva' }));
        break;
      case 'mssql':
        setForm(prev => ({ ...prev, dbType: val, host: '127.0.0.1', port: '1433', userName: 'mssql', dbName: 'gva' }));
        break;
      case 'sqlite':
        setForm(prev => ({ ...prev, dbType: val, host: '', port: '', userName: '', password: '', dbPath: '' }));
        break;
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setForm(prev => ({ ...prev, [name]: value }));
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (form.adminPassword.length < 6) {
      toast.error('管理员初始密码长度不能小于6位');
      return;
    }

    setIsLoading(true);
    const loadingToast = toast.loading('正在初始化数据库，请稍候...');

    try {
      const res: any = await initDB(form);
      if (res.code === 0) {
        toast.success(res.msg || '自动创建数据库成功', { id: loadingToast });
        navigate('/login');
      } else {
        toast.error(res.msg || '初始化失败', { id: loadingToast });
      }
    } catch (error) {
      console.error(error);
      toast.error('初始化请求失败', { id: loadingToast });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#F8FAFC] flex flex-col items-center justify-center p-4">
      {/* Container */}
      <div className="max-w-xl w-full bg-white rounded-2xl shadow-xl overflow-hidden border border-gray-100 relative">
        {/* Header Decor */}
        <div className="h-2 w-full bg-blue-600"></div>
        
        <div className="px-8 py-10">
          <div className="text-center mb-8">
            <h1 className="text-3xl font-extrabold text-gray-900 mb-2">系统初始化设置</h1>
            <p className="text-gray-500">连接您的数据库以开始使用平台服务</p>
          </div>

          <form onSubmit={onSubmit} className="space-y-5">
            <div className="grid grid-cols-2 gap-4">
               {/* 数据库类别 */}
               <div className="col-span-2 md:col-span-1">
                 <label className="block text-sm font-semibold text-gray-700 mb-1 flex items-center gap-1"><Database size={16}/> 数据库类型</label>
                 <select 
                   value={form.dbType} 
                   onChange={handleDBTypeChange}
                   className="w-full bg-gray-50 border border-gray-200 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block p-3 outline-none"
                 >
                   <option value="mysql">MySQL</option>
                   <option value="pgsql">PostgreSQL</option>
                   <option value="oracle">Oracle</option>
                   <option value="mssql">SQL Server</option>
                   <option value="sqlite">SQLite</option>
                 </select>
               </div>

               {/* Admin Default Pass */}
               <div className="col-span-2 md:col-span-1">
                 <label className="block text-sm font-semibold text-gray-700 mb-1 flex items-center gap-1"><ShieldCheck size={16}/> 超管初始密码</label>
                 <input 
                   type="text" 
                   name="adminPassword"
                   value={form.adminPassword} 
                   onChange={handleChange}
                   placeholder="admin账号的默认密码"
                   className="w-full bg-gray-50 border border-gray-200 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block p-3 outline-none"
                 />
               </div>
            </div>

            {form.dbType !== 'sqlite' && (
              <>
                <div className="grid grid-cols-5 gap-4">
                  <div className="col-span-3">
                    <label className="block text-sm font-semibold text-gray-700 mb-1 flex items-center gap-1"><Server size={16}/> 主机地址 (Host)</label>
                    <input type="text" name="host" value={form.host} onChange={handleChange} className="w-full bg-gray-50 border border-gray-200 rounded-lg p-3 outline-none text-sm focus:ring-blue-500 focus:border-blue-500" required />
                  </div>
                  <div className="col-span-2">
                    <label className="block text-sm font-semibold text-gray-700 mb-1">端口</label>
                    <input type="text" name="port" value={form.port} onChange={handleChange} className="w-full bg-gray-50 border border-gray-200 rounded-lg p-3 outline-none text-sm focus:ring-blue-500 focus:border-blue-500" required />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-1 flex items-center gap-1"><User size={16}/> 用户名</label>
                    <input type="text" name="userName" value={form.userName} onChange={handleChange} className="w-full bg-gray-50 border border-gray-200 rounded-lg p-3 outline-none text-sm focus:ring-blue-500 focus:border-blue-500" required />
                  </div>
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-1 flex items-center gap-1"><Key size={16}/> 密码</label>
                    <input type="text" name="password" value={form.password} onChange={handleChange} placeholder="(空)" className="w-full bg-gray-50 border border-gray-200 rounded-lg p-3 outline-none text-sm focus:ring-blue-500 focus:border-blue-500" />
                  </div>
                </div>
              </>
            )}

            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-1 flex items-center gap-1"><Database size={16}/> 数据库名称</label>
              <input type="text" name="dbName" value={form.dbName} onChange={handleChange} className="w-full bg-gray-50 border border-gray-200 rounded-lg p-3 outline-none text-sm focus:ring-blue-500 focus:border-blue-500" required />
            </div>

            {form.dbType === 'sqlite' && (
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1">SQLite存放路径 (dbPath)</label>
                <input type="text" name="dbPath" value={form.dbPath} onChange={handleChange} placeholder="请输入sqlite数据库文件存放绝对路径" className="w-full bg-gray-50 border border-gray-200 rounded-lg p-3 outline-none text-sm focus:ring-blue-500 focus:border-blue-500" required />
              </div>
            )}

            {form.dbType === 'pgsql' && (
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-1">PostgreSQL Template</label>
                <input type="text" name="template" value={form.template} onChange={handleChange} className="w-full bg-gray-50 border border-gray-200 rounded-lg p-3 outline-none text-sm focus:ring-blue-500 focus:border-blue-500" required />
              </div>
            )}

            <div className="pt-4 mt-8 flex flex-col sm:flex-row gap-3">
              <button 
                type="button"
                onClick={() => navigate('/login')}
                className="w-full sm:w-1/3 py-3 px-4 bg-white border border-gray-300 text-gray-700 rounded-xl font-medium hover:bg-gray-50 transition-colors"
                disabled={isLoading}
              >
                返回登录
              </button>
              <button 
                type="submit"
                disabled={isLoading}
                className="w-full sm:w-2/3 py-3 px-4 bg-blue-600 border border-transparent text-white rounded-xl font-medium hover:bg-blue-700 transition-colors flex items-center justify-center disabled:bg-blue-400"
              >
                {isLoading ? '配置中...' : '立即初始化'}
              </button>
            </div>

          </form>
        </div>
      </div>
    </div>
  );
}
