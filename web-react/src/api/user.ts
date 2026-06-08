import service from '../utils/request';

export const login = (data: any) => {
  return service({
    url: '/base/login',
    method: 'post',
    data,
  });
};

export const getUserInfo = () => {
  return service({
    url: '/user/getUserInfo',
    method: 'get',
  });
};

export const setSelfSetting = (data: Record<string, unknown>) => {
  return service({
    url: '/user/setSelfSetting',
    method: 'put',
    data,
  });
};

export const captcha = (data?: any) => {
  return service({
    url: '/base/captcha',
    method: 'post',
    data: data || {},
  });
};
