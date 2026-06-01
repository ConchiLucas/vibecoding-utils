import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

interface UserInfo {
  ID: number;
  uuid: string;
  userName: string;
  nickName: string;
  headerImg: string;
  authorityId: number;
  phone: string;
  email: string;
  [key: string]: any;
}

interface UserState {
  token: string;
  userInfo: UserInfo;
  setToken: (token: string) => void;
  setUserInfo: (info: UserInfo) => void;
  logout: () => void;
}

const defaultUserInfo: UserInfo = {
  ID: 0,
  uuid: '',
  userName: '',
  nickName: '',
  headerImg: '',
  authorityId: 0,
  phone: '',
  email: ''
}

export const useUserStore = create<UserState>()(
  persist(
    (set) => ({
      token: '',
      userInfo: defaultUserInfo,
      setToken: (token) => set({ token }),
      setUserInfo: (userInfo) => set({ userInfo }),
      logout: () => set({ token: '', userInfo: defaultUserInfo }),
    }),
    {
      name: 'easy-deploy-user-storage',
      storage: createJSONStorage(() => localStorage),
    }
  )
)
