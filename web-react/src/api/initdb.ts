import service from '../utils/request';

export const initDB = (data: any) => {
  return service({
    url: '/init/initdb',
    method: 'post',
    data,
    donNotShowLoading: true,
  });
};

export const checkDB = () => {
  return service({
    url: '/init/checkdb',
    method: 'post',
  });
};

export const migrateTables = () => {
  return service({
    url: '/init/migrateTables',
    method: 'post',
  });
};
