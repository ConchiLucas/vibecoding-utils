import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

interface ProjectState {
  activeProject: string;
  activeProjectId: number | null;
  activeConnectionId: number | null;
  setActiveProject: (name: string, id?: number) => void;
  setActiveConnectionId: (id: number | null) => void;
  hydrateActiveSelection: (selection: {
    activeProject?: string;
    activeProjectId?: number | null;
    activeConnectionId?: number | null;
  }) => void;
}

export const useProjectStore = create<ProjectState>()(
  persist(
    (set) => ({
      activeProject: '',
      activeProjectId: null,
      activeConnectionId: null,
      setActiveProject: (name, id) => set((state) => {
        const nextProjectId = id ?? null;
        const isSameProject =
          state.activeProjectId === nextProjectId ||
          (!state.activeProjectId && state.activeProject === name);
        return {
          activeProject: name,
          activeProjectId: nextProjectId,
          activeConnectionId: isSameProject ? state.activeConnectionId : null,
        };
      }),
      setActiveConnectionId: (id) => set({ activeConnectionId: id }),
      hydrateActiveSelection: (selection) => set((state) => ({
        activeProject: Object.prototype.hasOwnProperty.call(selection, 'activeProject')
          ? selection.activeProject || ''
          : state.activeProject,
        activeProjectId: Object.prototype.hasOwnProperty.call(selection, 'activeProjectId')
          ? selection.activeProjectId ?? null
          : state.activeProjectId,
        activeConnectionId: Object.prototype.hasOwnProperty.call(selection, 'activeConnectionId')
          ? selection.activeConnectionId ?? null
          : state.activeConnectionId,
      })),
    }),
    {
      name: 'easy-test-active-project',
      storage: createJSONStorage(() => localStorage),
    }
  )
)
