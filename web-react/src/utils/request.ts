import axios, { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse, AxiosError } from 'axios';
import { useUserStore } from '../stores/useUserStore';
import toast from 'react-hot-toast';

// Extending AxiosRequestConfig to support custom properties
declare module 'axios' {
  export interface AxiosRequestConfig {
    donNotShowLoading?: boolean;
    loadingOption?: any;
  }
}

const service: AxiosInstance = axios.create({
  timeout: 99999,
  baseURL: import.meta.env.VITE_BASE_API || '/api',
});

// A simple loading placeholder log/toast or integrating NProgress later
let activeAxios = 0;

const showLoading = () => {
  activeAxios++;
  // You can integrate nprogress here
};

const closeLoading = () => {
  activeAxios--;
  if (activeAxios <= 0) {
    activeAxios = 0;
  }
};

service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (!config.donNotShowLoading) {
      showLoading();
    }
    
    // baseURL 已在 axios.create 时根据环境正确设置
    
    // Read state outside React component 
    const { token, userInfo } = useUserStore.getState();
    
    if (config.data instanceof FormData) {
      config.headers.delete('Content-Type');
    } else if (!config.headers.has('Content-Type')) {
      config.headers.set('Content-Type', 'application/json');
    }
    if (token) {
      config.headers.set('x-token', token);
    }
    if (userInfo?.ID) {
      config.headers.set('x-user-id', String(userInfo.ID));
    }
    
    return config;
  },
  (error: AxiosError) => {
    if (!error.config?.donNotShowLoading) {
      closeLoading();
    }
    toast.error(error.message || '请求发送失败');
    return Promise.reject(error);
  }
);

function getErrorMessage(error: any) {
  return error.response?.data?.msg || error.response?.statusText || '请求失败';
}

service.interceptors.response.use(
  (response: AxiosResponse) => {
    if (!response.config.donNotShowLoading) {
      closeLoading();
    }

    const setToken = useUserStore.getState().setToken;
    
    if (response.headers['new-token']) {
      setToken(response.headers['new-token']);
    }

    if (typeof response.data.code === 'undefined') {
      return response;
    }

    // Success logical resolution from old vue app
    if (response.data.code === 0 || response.headers.success === 'true') {
      if (response.headers.msg) {
        response.data.msg = decodeURI(response.headers.msg as string);
      }
      return response.data;
    } else {
      toast.error(response.data.msg || decodeURI(response.headers.msg as string || '请求异常'));
      return response.data.msg ? response.data : response;
    }
  },
  (error: any) => {
    if (!error.config?.donNotShowLoading) {
      closeLoading();
    }

    if (!error.response) {
      toast.error(getErrorMessage(error));
      return Promise.reject(error);
    }

    if (error.response.status === 401) {
      toast.error('认证失败，请重新登录');
      useUserStore.getState().logout();
      window.location.href = '/login'; // quick redirect for react
      return Promise.reject(error);
    }

    toast.error(getErrorMessage(error));
    return Promise.reject(error);
  }
);

export default service;
