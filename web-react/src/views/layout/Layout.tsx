import React from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import Header from './Header';
import AIChatWidget from '../../components/AIChatWidget/AIChatWidget';

export default function Layout() {
  const location = useLocation();
  const isFullWidthRoute = 
    location.pathname === '/projects' || 
    location.pathname === '/script-manager' || 
    location.pathname === '/agile-request' || 
    location.pathname === '/agile-table-samples' ||
    location.pathname.includes('/templates');

  return (
    <div className="min-h-screen bg-[#FAFAFA] text-gray-900 font-sans flex flex-col transition-colors duration-300">
      {/* Top Navbar */}
      {!location.pathname.includes('/templates') && <Header />}

      {/* Page Content Wrapper */}
      <div className="w-full flex-1 flex flex-col">
        <main className={`w-full flex-1 transition-all duration-300 ${isFullWidthRoute ? '' : 'px-4 sm:px-6 lg:px-8 py-8'}`}>
          <Outlet />
        </main>
      </div>

      {/* Minimal Footer */}
      {!location.pathname.includes('/templates') && (
        <footer className="border-t border-gray-100 py-6 text-center text-sm text-gray-400">
          <p>由 Easy Deploy 提供运行时支持 · 极简部署自动化流水线</p>
        </footer>
      )}

      {/* AI Chat Floating Widget */}
      <AIChatWidget />
    </div>
  );
}
