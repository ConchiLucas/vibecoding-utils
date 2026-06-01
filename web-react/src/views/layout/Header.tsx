import React, { useState, useRef, useEffect } from 'react';
import { NavLink, useNavigate, useLocation } from 'react-router-dom';
import { useUserStore } from '../../stores/useUserStore';
import { useProjectStore } from '../../stores/useProjectStore';
import { CloudLightning, Terminal, LogOut, Key, ArrowLeft, User, Route, GitMerge, Settings, FileCode, Moon, Sun, FolderOpen, Sparkles, Send, Database } from 'lucide-react';
import clsx from 'clsx';
import toast from 'react-hot-toast';

export default function Header() {
  const userInfo = useUserStore((state) => state.userInfo);
  const activeProject = useProjectStore((state) => state.activeProject);
  const logout = useUserStore((state) => state.logout);
  const navigate = useNavigate();
  const location = useLocation();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const [isDark, setIsDark] = useState(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('theme') === 'dark' || 
        (!localStorage.getItem('theme') && window.matchMedia('(prefers-color-scheme: dark)').matches);
    }
    return false;
  });

  useEffect(() => {
    const root = window.document.documentElement;
    if (isDark) {
      root.classList.add('chrome-dark');
      root.classList.remove('dark');
      localStorage.setItem('theme', 'dark');
    } else {
      root.classList.remove('chrome-dark');
      localStorage.setItem('theme', 'light');
    }
  }, [isDark]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleLogout = () => {
    logout();
    toast.success('已安全退出登陆');
    navigate('/login');
  };

  const navLinks = [
    { name: '项目池', path: '/projects', icon: CloudLightning },
    { name: '脚本库', path: '/script-manager', icon: Terminal },
    { name: '接口转发', path: '/interfaces', icon: Route },
    { name: '表关系', path: '/keywords', icon: GitMerge },
    { name: '代码生成', path: '/code-generate', icon: FileCode },
    { name: '配置管理', path: '/config', icon: Settings },
    { name: '表样本', path: '/agile-table-samples', icon: Database },
    { name: '敏捷请求', path: '/agile-request', icon: Send },
  ];

  const isScriptRoute = location.pathname.includes('/scripts');

  return (
    <header className="sticky top-0 z-50 w-full bg-white/80 backdrop-blur-md border-b border-gray-100 transition-colors duration-300">
      <div className="w-full px-4 sm:px-6 lg:px-8 relative h-16 flex items-center justify-between">
        
        {/* Brand & Main Navigation (Left) */}
        <div className="flex items-center space-x-4 z-10">
          {/* 脚本页返回按钮 */}
          {isScriptRoute && (
            <button
              onClick={() => navigate('/projects')}
              className="flex items-center gap-1 text-sm font-medium text-gray-500 hover:text-gray-900 transition-colors px-2 py-1.5 rounded-lg hover:bg-gray-50"
            >
              <ArrowLeft size={16} />
            </button>
          )}

          <div className="flex items-center gap-2 cursor-pointer transition-opacity hover:opacity-80" onClick={() => navigate('/projects')}>
            <div className="w-8 h-8 rounded-lg bg-black flex items-center justify-center shadow-md">
               <span className="text-white font-extrabold text-sm select-none">ED</span>
            </div>
            <span className="font-bold text-gray-900 tracking-tight text-xl">VibeDeploy</span>
          </div>

          <nav className="hidden md:flex items-center gap-1 min-w-0">
            {navLinks.map((item) => {
              const active = location.pathname.startsWith(item.path);
              return (
                <NavLink
                  key={item.path}
                  to={item.path}
                  className={clsx(
                    "px-2 lg:px-3 xl:px-4 py-2 rounded-lg text-xs lg:text-sm font-medium transition-colors flex items-center gap-2 whitespace-nowrap shrink-0",
                    active 
                      ? "bg-gray-100 text-gray-900" 
                      : "text-gray-500 hover:bg-gray-50 hover:text-gray-900"
                  )}
                >
                  <item.icon size={16} className={clsx(active ? "opacity-100" : "opacity-70")} />
                  {item.name}
                </NavLink>
              );
            })}
          </nav>
        </div>

        {/* User Profile & Context Actions (Right) */}
        <div className="flex items-center gap-3 z-10">
          {/* Active Project Badge */}
          {activeProject && (
            <div
              onClick={() => navigate('/config')}
              title="点击管理项目配置"
              className="hidden sm:flex items-center gap-1.5 px-3 py-1.5 rounded-full border border-indigo-200 bg-indigo-50 hover:bg-indigo-100 cursor-pointer transition-colors group"
            >
              <FolderOpen size={13} className="text-indigo-500 shrink-0" />
              <span className="text-[11px] text-indigo-400 font-semibold tracking-wide uppercase leading-none">项目</span>
              <span className="text-xs font-bold text-indigo-700 max-w-[120px] truncate leading-none">{activeProject}</span>
            </div>
          )}
          {/* Theme Toggle */}
          <button
            onClick={() => setIsDark(!isDark)}
            className="p-2 text-gray-500 hover:text-gray-900 hover:bg-gray-100 rounded-full transition-colors hidden sm:flex items-center justify-center"
            title={isDark ? "切换明亮模式" : "切换暗黑模式"}
          >
            {isDark ? <Sun size={18} /> : <Moon size={18} />}
          </button>

          {/* AI Chat Trigger */}
          <button
            onClick={() => window.dispatchEvent(new CustomEvent('toggle-ai-chat'))}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-full text-white text-xs font-semibold transition-all duration-200 hover:shadow-lg hover:shadow-indigo-500/25 hover:-translate-y-0.5 active:scale-95"
            style={{ background: 'linear-gradient(135deg, #6366f1, #8b5cf6)' }}
            title="AI 助手"
          >
            <Sparkles size={14} />
            <span>AI</span>
          </button>

          {/* Profile Dropdown */}
          <div className="relative" ref={dropdownRef}>
            <div 
              onClick={() => setDropdownOpen(!dropdownOpen)}
              className="flex items-center cursor-pointer p-1.5 hover:bg-gray-50 rounded-full border border-transparent hover:border-gray-200 transition-all select-none"
            >
              <div className="w-8 h-8 rounded-full bg-slate-100 flex items-center justify-center text-slate-600 shadow-sm border border-slate-200">
                <User size={16} strokeWidth={2.5} />
              </div>
            </div>

            {dropdownOpen && (
              <div className="absolute right-0 mt-3 w-48 bg-white rounded-xl shadow-xl border border-gray-100 py-1.5 animate-in fade-in slide-in-from-top-2 duration-200 z-50">
                <div className="px-4 py-2 border-b border-gray-50 mb-1">
                  <p className="text-xs text-gray-400 font-semibold tracking-wider">当前账号</p>
                  <p className="text-sm font-medium text-gray-900 truncate mt-0.5">{userInfo?.userName || 'admin'}</p>
                </div>
                <button 
                  onClick={() => { setDropdownOpen(false); toast('修改密码功能待接入开发'); }}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 hover:text-black transition-colors"
                >
                  <Key size={16} className="text-gray-400" /> 修改密码
                </button>
                <div className="h-px bg-gray-100 my-1"></div>
                <button 
                  onClick={handleLogout}
                  className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 transition-colors"
                >
                  <LogOut size={16} className="text-red-400" /> 退出登陆
                </button>
              </div>
            )}
          </div>
        </div>

      </div>
    </header>
  );
}
