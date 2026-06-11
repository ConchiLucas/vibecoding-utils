import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { useUserStore } from './stores/useUserStore';

import Login from './views/login/Login';
import Layout from './views/layout/Layout';
import ProjectDashboard from './views/project/ProjectDashboard';
import CodeGenerateDashboard from './views/code_generate/ProjectDashboard';
import DbTemplateLibrary from './views/code_generate/DbTemplateLibrary';
import CodeGenerateTemplates from './views/code_generate/ProjectTemplates';
import ProjectScripts from './views/project/ProjectScripts';
import ScriptManager from './views/script-manager/ScriptManager';
import ServerManager from './views/server/ServerManager';
import Init from './views/init/Init';

import DictManager from './views/dict-manager/DictManager';
import InterfaceServerManager from './views/server-manager/ServerManager';
import InterfaceManager from './views/interface-manager/InterfaceManager';
import ConnectionManager from './views/connection-manager/ConnectionManager';

import TableManager from './views/table-manager/TableManager';
import TableColumnManager from './views/table-column-manager/TableColumnManager';
import RelateManager from './views/relate-manager/RelateManager';
import EntityManager from './views/entity-manager/EntityManager';
import EntityColumnManager from './views/entity-column-manager/EntityColumnManager';

import ClientManager from './views/client-manager/ClientManager';
import InterfaceParamsManager from './views/interface-params-manager/InterfaceParamsManager';
import InterfaceLogManager from './views/interface-log-manager/InterfaceLogManager';
import PreferManager from './views/prefer-manager/PreferManager';
import KeywordsManager from './views/keywords/KeywordsManager';
import ConfigManager from './views/config-manager/ConfigManager';
import AgileRequestManager from './views/agile-request/AgileRequestManager';
import AgileTableSamplesManager from './views/agile-table-samples/AgileTableSamplesManager';
import LogManager from './views/log-manager/LogManager';
import DevelopmentPrepareManager from './views/development-prepare/DevelopmentPrepareManager';

// A simple protective wrapper for authenticated routes
const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const token = useUserStore((state) => state.token);
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
};

function App() {
  return (
    <BrowserRouter>
      {/* Toast Notification Provider */}
      <Toaster position="top-center"
        containerStyle={{
          zIndex: 20000,
        }}
        toastOptions={{
          style: {
            borderRadius: '12px',
            background: '#333',
            color: '#fff',
          },
        }}
      />
      
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/init" element={<Init />} />
        
        {/* Main Application Routes Wrapped in Layout */}
        <Route 
          path="/" 
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          } 
        >
          {/* Default redirect inside layout */}
          <Route index element={<Navigate to="/projects" replace />} />
          <Route path="projects" element={<ProjectDashboard />} />
          <Route path="projects/:projectId/scripts" element={<ProjectScripts />} />
          <Route path="servers" element={<ServerManager />} />
          <Route path="script-manager" element={<ScriptManager />} />
          
          {/* 代码生成 */}
          <Route path="code-generate" element={<CodeGenerateDashboard />} />
          <Route path="code-generate/:projectId/db-templates" element={<DbTemplateLibrary />} />
          <Route path="code-generate/:projectId/templates" element={<CodeGenerateTemplates />} />
          
          {/* Imported from go-easy-test */}
          <Route path="dicts" element={<DictManager />} />
          <Route path="interface-servers" element={<InterfaceServerManager />} />
          <Route path="interfaces" element={<InterfaceManager />} />
          <Route path="connections" element={<ConnectionManager />} />
          <Route path="tables" element={<TableManager />} />
          <Route path="table-columns" element={<TableColumnManager />} />
          <Route path="relates" element={<RelateManager />} />
          <Route path="entities" element={<EntityManager />} />
          <Route path="entity-columns" element={<EntityColumnManager />} />
          <Route path="clients" element={<ClientManager />} />
          <Route path="interface-params" element={<InterfaceParamsManager />} />
          <Route path="interface-logs" element={<InterfaceLogManager />} />
          <Route path="prefers" element={<PreferManager />} />
          <Route path="keywords" element={<KeywordsManager />} />
          <Route path="config" element={<ConfigManager />} />
          <Route path="agile-table-samples" element={<AgileTableSamplesManager />} />
          <Route path="agile-request" element={<AgileRequestManager />} />
          <Route path="log-manager" element={<LogManager />} />
          <Route path="development-prepare" element={<DevelopmentPrepareManager />} />
        </Route>

        {/* Fallback */}
        <Route path="*" element={<Navigate to="/projects" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
