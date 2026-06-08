import { useEffect, useRef, useState } from 'react';
import { getUserInfo, setSelfSetting } from '../api/user';
import { useProjectStore } from '../stores/useProjectStore';
import { useUserStore } from '../stores/useUserStore';
import {
  extractRemoteProjectSelection,
  hasCompleteProjectSelection,
  normalizeProjectSelection,
  ProjectSelection,
  resolveProjectSelection,
  selectionEquals,
} from '../utils/projectSelection';

const ACTIVE_SELECTION_KEY = 'activeSelection';

export function usePersistedProjectSelection() {
  const token = useUserStore(state => state.token);
  const userInfo = useUserStore(state => state.userInfo);
  const setUserInfo = useUserStore(state => state.setUserInfo);
  const activeProject = useProjectStore(state => state.activeProject);
  const activeProjectId = useProjectStore(state => state.activeProjectId);
  const activeConnectionId = useProjectStore(state => state.activeConnectionId);
  const hydrateActiveSelection = useProjectStore(state => state.hydrateActiveSelection);
  const [hydrationVersion, setHydrationVersion] = useState(0);
  const hydratedRef = useRef(false);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastSavedRef = useRef('');

  useEffect(() => {
    if (!token || hydratedRef.current) return;

    let cancelled = false;
    const hydrate = async () => {
      const localSelection = normalizeProjectSelection(useProjectStore.getState());
      let remoteSelection = extractRemoteProjectSelection(userInfo?.originSetting);
      let latestUserInfo = userInfo;

      if (!hasCompleteProjectSelection(localSelection)) {
        try {
          const res: any = await getUserInfo();
          latestUserInfo = res?.data?.userInfo || latestUserInfo;
          if (latestUserInfo) {
            setUserInfo(latestUserInfo);
          }
          remoteSelection = extractRemoteProjectSelection(latestUserInfo?.originSetting);
        } catch {
          remoteSelection = extractRemoteProjectSelection(userInfo?.originSetting);
        }
      }

      if (cancelled) return;

      const nextSelection = resolveProjectSelection(localSelection, remoteSelection);
      if (!selectionEquals(localSelection, nextSelection)) {
        hydrateActiveSelection(nextSelection);
      }
      hydratedRef.current = true;
      lastSavedRef.current = JSON.stringify(remoteSelection);
      setHydrationVersion(version => version + 1);
    };

    void hydrate();

    return () => {
      cancelled = true;
    };
  }, [hydrateActiveSelection, setUserInfo, token, userInfo]);

  useEffect(() => {
    if (!token || !hydratedRef.current) return;

    const selection: ProjectSelection = {
      activeProject,
      activeProjectId,
      activeConnectionId,
    };
    const serialized = JSON.stringify(selection);
    if (serialized === lastSavedRef.current) return;

    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current);
    }

    saveTimerRef.current = setTimeout(() => {
      void setSelfSetting({
        [ACTIVE_SELECTION_KEY]: selection,
      }).then(() => {
        const currentUserInfo = useUserStore.getState().userInfo;
        useUserStore.getState().setUserInfo({
          ...currentUserInfo,
          originSetting: {
            ...(currentUserInfo?.originSetting || {}),
            [ACTIVE_SELECTION_KEY]: selection,
          },
        });
        lastSavedRef.current = serialized;
      }).catch(() => {
        // Selection is still kept in localStorage by zustand; DB sync can retry on the next change.
      });
    }, 350);

    return () => {
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
    };
  }, [activeConnectionId, activeProject, activeProjectId, hydrationVersion, token]);
}
