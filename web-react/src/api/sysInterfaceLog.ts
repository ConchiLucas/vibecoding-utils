import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbInterfaceLog {
  ID: number;
  interfacePaths: string;
  userName: string;
  isSuccess: number;
  reqParams: string;
  resParams: string;
  environment: string;
  identity: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbInterfaceLog = (data: Partial<TbInterfaceLog>) => {
  return service.post<any, ApiResponse>('/interfaceLog/createTbInterfaceLog', data);
};

export const deleteTbInterfaceLog = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/interfaceLog/deleteTbInterfaceLog', { data });
};

export const updateTbInterfaceLog = (data: Partial<TbInterfaceLog>) => {
  return service.put<any, ApiResponse>('/interfaceLog/updateTbInterfaceLog', data);
};

export const getTbInterfaceLogList = (params: { page: number; pageSize: number; interfacePaths?: string; environment?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbInterfaceLog>>>('/interfaceLog/getTbInterfaceLogList', { params });
};

export interface LogParamsPreview {
  paramsName: string;
  content: string;
}

export const getParamsPreview = (params: { ID: number }) => {
  return service.get<any, ApiResponse<LogParamsPreview[]>>('/interfaceLog/getParamsPreview', { params });
};
