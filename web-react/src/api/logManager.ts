import service from '../utils/request';

export interface LogProjectRoute {
  ID: number;
  projectId: number;
  routeKey?: string;
  routeName?: string;
  localProjectPath?: string;
  localExecuteCommand?: string;
  localStartCommand?: string;
  localStopCommand?: string;
  logFilePath?: string;
  buildType?: string;
  dockerComposeDeploy?: boolean;
  color?: string;
  icon?: string;
  sort?: number;
  routeType?: 'script' | 'file_log' | 'docker_compose';
}

export interface LogProject {
  ID: number;
  groupId?: number;
  projectConfigId?: number;
  projectConfigName?: string;
  projectName: string;
  description?: string;
  computerLanguage?: string;
  localProjectPath?: string;
  routes?: LogProjectRoute[];
}

export interface LogProjectGroup {
  ID: number;
  groupName: string;
  sort?: number;
}

export interface DockerServiceSummary {
  projectId: number;
  routeId: number;
  routeName: string;
  serviceName: string;
  workDir: string;
  source: string;
  logFilePath?: string;
  routeType?: 'script' | 'file_log' | 'docker_compose';
  running?: boolean;
}

export const getLogProjectPage = (data: any) => {
  return service({
    url: '/logManager/page',
    method: 'post',
    data,
    donNotShowLoading: true,
  });
};

export const getLogProjectGroups = () => {
  return service({
    url: '/logManager/groups',
    method: 'get',
    donNotShowLoading: true,
  });
};

export const saveOrUpdateLogProject = (data: any) => {
  return service({
    url: '/logManager/saveOrUpdateProject',
    method: 'post',
    data,
  });
};

export const deleteLogProject = (ids: number | string) => {
  return service({
    url: `/logManager/deleteProject/${ids}`,
    method: 'delete',
  });
};

export const saveOrUpdateLogRoute = (data: any) => {
  return service({
    url: '/logManager/saveOrUpdateRoute',
    method: 'post',
    data,
  });
};

export const deleteLogRoute = (id: number | string) => {
  return service({
    url: `/logManager/deleteRoute/${id}`,
    method: 'delete',
  });
};

export const getLogManagerDockerServices = (projectId: number | string, scope?: 'service' | 'docker') => {
  return service({
    url: `/logManager/dockerServices/${projectId}`,
    method: 'get',
    params: scope ? { scope } : undefined,
    donNotShowLoading: true,
  });
};
