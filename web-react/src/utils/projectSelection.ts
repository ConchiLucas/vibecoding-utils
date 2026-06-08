export type ProjectSelection = {
  activeProject: string;
  activeProjectId: number | null;
  activeConnectionId: number | null;
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const toNullableNumber = (value: unknown): number | null => {
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) return value;
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
  }
  return null;
};

export const normalizeProjectSelection = (value: unknown): ProjectSelection => {
  const source = isRecord(value) ? value : {};
  return {
    activeProject: typeof source.activeProject === 'string' ? source.activeProject : '',
    activeProjectId: toNullableNumber(source.activeProjectId),
    activeConnectionId: toNullableNumber(source.activeConnectionId),
  };
};

export const hasCompleteProjectSelection = (selection: ProjectSelection) =>
  Boolean(selection.activeProject || selection.activeProjectId) && Boolean(selection.activeConnectionId);

export const selectionEquals = (a: ProjectSelection, b: ProjectSelection) =>
  a.activeProject === b.activeProject &&
  a.activeProjectId === b.activeProjectId &&
  a.activeConnectionId === b.activeConnectionId;

export const resolveProjectSelection = (
  localSelection: ProjectSelection,
  remoteSelection: ProjectSelection
): ProjectSelection => {
  const hasLocalProject = Boolean(localSelection.activeProject || localSelection.activeProjectId);
  const projectMatchesRemote =
    !hasLocalProject ||
    !remoteSelection.activeProjectId ||
    localSelection.activeProjectId === remoteSelection.activeProjectId;

  return {
    activeProject: localSelection.activeProject || remoteSelection.activeProject,
    activeProjectId: localSelection.activeProjectId ?? remoteSelection.activeProjectId,
    activeConnectionId:
      localSelection.activeConnectionId ??
      (projectMatchesRemote ? remoteSelection.activeConnectionId : null),
  };
};

export const extractRemoteProjectSelection = (originSetting: unknown) => {
  if (!isRecord(originSetting)) return normalizeProjectSelection(null);
  return normalizeProjectSelection(originSetting.activeSelection);
};
