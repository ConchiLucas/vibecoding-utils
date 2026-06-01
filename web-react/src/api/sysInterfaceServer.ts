import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbInterfaceServer {
  ID: number;
  projectName: string;
  serverName: string;
  userName: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbInterfaceServer = (data: Partial<TbInterfaceServer>) => {
  return service.post<any, ApiResponse>('/server/createTbInterfaceServer', data);
};

export const deleteTbInterfaceServer = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/server/deleteTbInterfaceServer', { data });
};

export const updateTbInterfaceServer = (data: Partial<TbInterfaceServer>) => {
  return service.put<any, ApiResponse>('/server/updateTbInterfaceServer', data);
};

export const getTbInterfaceServerList = (params: { page: number; pageSize: number; projectName?: string; serverName?: string; userName?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbInterfaceServer>>>('/server/getTbInterfaceServerList', { params });
};

export const uploadSwaggerJson = (formData: FormData) => {
  return service.put<any, ApiResponse>('/server/upload', formData, {
    timeout: 60000,
  });
};

export interface TreeNode {
  id: number;
  pid?: number;
  projectName?: string;
  serverName?: string;
  interfaceName: string;
  children?: TreeNode[];
}

export const buildServerTree = (projectName?: string) => {
  return service.post<any, ApiResponse<TreeNode[]>>('/server/buildTree', { projectName: projectName || '' });
};

export const renameServer = (data: { ID: number; serverName: string }) => {
  return service.put<any, ApiResponse>('/server/renameServer', data);
};
