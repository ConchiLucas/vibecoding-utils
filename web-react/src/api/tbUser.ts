import service from '../utils/request';
import { ApiResponse, PageResult } from './sysDict';

export interface TbUser {
  ID: number;
  userName: string;
  nickName: string;
  headerImg: string;
  phone: string;
  email: string;
  enable: number;
  roleId?: number;
  CreatedAt?: string;
}

export const getUserList = (data: { page: number; pageSize: number }) => {
  return service.post<any, ApiResponse<PageResult<TbUser>>>('/user/getUserList', data);
};

export const deleteUser = (data: { id: number }) => {
  return service.delete<any, ApiResponse>('/user/deleteUser', { data });
};

export const registerUser = (data: Partial<TbUser> & { password?: string }) => {
  return service.post<any, ApiResponse>('/user/admin_register', data);
};

export const setUserInfo = (data: Partial<TbUser>) => {
  return service.put<any, ApiResponse>('/user/setUserInfo', data);
};
