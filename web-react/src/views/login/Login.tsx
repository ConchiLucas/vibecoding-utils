import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { User, Lock } from 'lucide-react';
import toast from 'react-hot-toast';
import { login } from '../../api/user';
import { checkDB, migrateTables } from '../../api/initdb';
import { useUserStore } from '../../stores/useUserStore';
import ConfirmDialog from '../../components/ConfirmDialog';
import { useConfirm } from '../../hooks/useConfirm';
import './login.css';

export default function Login() {
  const navigate = useNavigate();
  const setToken = useUserStore((state) => state.setToken);
  const setUserInfo = useUserStore((state) => state.setUserInfo);

  const [formData, setFormData] = useState({
    username: 'admin',
    password: '123456', // 默认密码通常为 123456
  });

  const [isLoading, setIsLoading] = useState(false);
  const { confirm, dialogProps } = useConfirm();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (formData.username.length < 5) {
      toast.error('请输入正确的用户名(最少5位)');
      return;
    }
    if (formData.password.length < 6) {
      toast.error('请输入正确的密码(最少6位)');
      return;
    }

    setIsLoading(true);
    const loadingToast = toast.loading('登录中，请稍候...');
    
    try {
      const res: any = await login(formData);
      if (res.code === 0) {
        setToken(res.data.token);
        setUserInfo(res.data.user);
        
        toast.success('登录成功', { id: loadingToast });
        navigate('/projects'); 
      } else {
        toast.error(res.msg || '登录失败', { id: loadingToast });
      }
    } catch (error) {
      console.error(error);
      toast.error('登录异常', { id: loadingToast });
    } finally {
      setIsLoading(false);
    }
  };

  const checkInit = async () => {
    try {
      const res: any = await checkDB();
      if (res.code !== 0) return;

      const { needInit, needMigrate } = res.data || {};

      if (needMigrate) {
        const ok = await confirm('检测到有新增的业务表尚未创建，是否立即同步表结构？');
        if (ok) {
          const migrateRes: any = await migrateTables();
          if (migrateRes.code === 0) {
            toast.success('同步表结构成功！');
          } else {
            toast.error(migrateRes.msg || '同步失败');
          }
        }
        return;
      }

      if (!needInit) {
        const ok = await confirm('已存在初始化配置，重新初始化将覆盖现有数据库配置。是否继续？');
        if (ok) {
          navigate('/init');
        }
        return;
      }

      navigate('/init');
    } catch (e: any) {
      toast.error('请求探测失败，请检查后端运行状态');
    }
  };

  return (
    <div className="premium-login-container">
      {/* Animated flowing gradient background */}
      <div className="animated-bg"></div>

      {/* Gentle Floating Orbs */}
      <div className="shape shape-1"></div>
      <div className="shape shape-2"></div>

      {/* Center Login Box: Solid White Card */}
      <div className="login-wrapper">
        <div className="solid-white-card relative z-10 w-full max-w-md mx-auto bg-white rounded-2xl p-10 shadow-[0_25px_50px_-12px_rgba(0,0,0,0.5)] border border-gray-200 flex flex-col items-center">
          <div className="brand-section text-center mb-8 w-full">
            <h2 className="title text-gray-900 text-[26px] font-extrabold mb-2">部署控制台</h2>
            <p className="subtitle text-gray-500 text-sm m-0">欢迎使用，请登录你的账号</p>
          </div>

          <form onSubmit={handleLogin} className="login-form w-full">
            <div className="mb-5 relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <User className="text-gray-400" size={18} />
              </div>
              <input
                type="text"
                name="username"
                value={formData.username}
                onChange={handleChange}
                className="light-theme-input w-full pl-10 pr-4 py-3 rounded-xl bg-gray-50 border-0 outline-none transition-all duration-300 text-gray-900 text-sm focus:bg-white focus:ring-2 focus:ring-blue-500"
                placeholder="用户名"
                required
              />
            </div>

            <div className="mb-8 relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Lock className="text-gray-400" size={18} />
              </div>
              <input
                type="password"
                name="password"
                value={formData.password}
                onChange={handleChange}
                className="light-theme-input w-full pl-10 pr-4 py-3 rounded-xl bg-gray-50 border-0 outline-none transition-all duration-300 text-gray-900 text-sm focus:bg-white focus:ring-2 focus:ring-blue-500"
                placeholder="密码"
                required
              />
            </div>

            <div className="mb-4">
              <button
                type="submit"
                disabled={isLoading}
                className="premium-button w-full py-4 rounded-xl text-white font-semibold flex items-center justify-center relative overflow-hidden transition-all duration-200"
              >
                <span>{isLoading ? '登录中...' : '登 录'}</span>
                <div className="btn-glow"></div>
              </button>
            </div>

            <div className="mb-0">
              <button
                type="button"
                onClick={checkInit}
                className="dev-button w-full py-[14px] rounded-xl bg-gray-100 text-gray-600 font-medium hover:bg-gray-200 hover:text-gray-900 transition-all duration-200 mt-2"
              >
                系统初始化设定
              </button>
            </div>
          </form>
        </div>
      </div>
      <ConfirmDialog {...dialogProps} />
    </div>
  );
}
