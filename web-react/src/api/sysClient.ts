import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbClient {
  ID: number;
  loginName: string;
  password?: string;
  nickName: string;
  userExtendParams: string;
  environment: string;
  identity: string;
  requestDemo: string;
  requestExtendParams: string;
  enableFlag: number;
  interfaceRequestHeader: string;
  remark: string;
  userName: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbClient = (data: Partial<TbClient>) => {
  return service.post<any, ApiResponse>('/client/createTbClient', data);
};

export const deleteTbClient = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/client/deleteTbClient', { data });
};

export const updateTbClient = (data: Partial<TbClient>) => {
  return service.put<any, ApiResponse>('/client/updateTbClient', data);
};

export const getTbClientList = (params: { page: number; pageSize: number; loginName?: string; nickName?: string; environment?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbClient>>>('/client/getTbClientList', { params });
};
