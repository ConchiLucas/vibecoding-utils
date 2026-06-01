import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbServerUser {
  ID: number;
  projectName: string;
  loginAccount: string;
  loginPassword: string;
  userNickname: string;
  roleCode: string;
  roleName: string;
  environment: string;
  requestHeader: string;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export const createTbServerUser = (data: Partial<TbServerUser>) => {
  return service.post<any, ApiResponse>('/serverUser/createTbInterfaceServerUser', data);
};

export const deleteTbServerUser = (data: { ID: number }) => {
  return service.delete<any, ApiResponse>('/serverUser/deleteTbInterfaceServerUser', { data });
};

export const updateTbServerUser = (data: Partial<TbServerUser>) => {
  return service.put<any, ApiResponse>('/serverUser/updateTbInterfaceServerUser', data);
};

export const getTbServerUserList = (params: { page: number; pageSize: number; projectName?: string; environment?: string; loginAccount?: string }) => {
  return service.get<any, ApiResponse<PageResult<TbServerUser>>>('/serverUser/getTbInterfaceServerUserList', { params });
};


export const updateClientStatus = (data: { ID: number; enableFlag: number }) => {
  return service.post<any, ApiResponse>('/serverUser/updateClientStatus', data);
};
