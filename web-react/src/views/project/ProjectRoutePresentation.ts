const LOCAL_FULL_ROUTE_COLOR = 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200';
const LOCAL_INCREMENTAL_ROUTE_COLOR = 'bg-sky-100 text-sky-700 hover:bg-sky-200';
const REMOTE_ROUTE_COLOR = 'bg-purple-100 text-purple-700 hover:bg-purple-200';
const DEPENDENCY_INCREMENTAL_ROUTE_COLOR = 'bg-amber-100 text-amber-800 hover:bg-amber-200';

export {
  LOCAL_FULL_ROUTE_COLOR,
  LOCAL_INCREMENTAL_ROUTE_COLOR,
  REMOTE_ROUTE_COLOR,
  DEPENDENCY_INCREMENTAL_ROUTE_COLOR,
};

function isDockerComposeRoute(route: any) {
  const routeKey = String(route?.routeKey || '').toLowerCase();
  const routeName = String(route?.routeName || '').toLowerCase();
  const buildType = String(route?.buildType || '').toLowerCase();
  return routeKey.includes('compose') || routeName.includes('compose') || buildType === 'docker_compose_deploy';
}

export function getRouteColor(route: any) {
  const routeKey = String(route?.routeKey || '').toLowerCase();
  const routeName = String(route?.routeName || '');
  if (route?.serverId || routeKey.includes('remote') || routeName.includes('远程')) {
    return REMOTE_ROUTE_COLOR;
  }
  if (routeName.includes('依赖增量')) {
    return DEPENDENCY_INCREMENTAL_ROUTE_COLOR;
  }
  if (routeKey.includes('incremental') || routeName.includes('增量')) {
    return LOCAL_INCREMENTAL_ROUTE_COLOR;
  }
  if (routeKey.includes('full') || routeName.includes('全量')) {
    return LOCAL_FULL_ROUTE_COLOR;
  }
  if (isDockerComposeRoute(route)) {
    return LOCAL_INCREMENTAL_ROUTE_COLOR;
  }
  return LOCAL_FULL_ROUTE_COLOR;
}

export function normalizeRouteColor(color: string | undefined, route: any) {
  const raw = String(color || '').trim();
  if (!raw) return getRouteColor(route);
  const buildType = String(route?.buildType || '').toLowerCase();
  const routeName = String(route?.routeName || '');
  if (routeName.includes('依赖增量') || routeName.includes('全量') || routeName.includes('增量') || buildType === 'docker_compose_deploy') return getRouteColor(route);
  if (raw.includes('rose') || raw.includes('pink')) return getRouteColor(route);
  return raw;
}

export function shouldExposeStopButton(route: any) {
  return getDisplayStopCommand(route) !== '';
}

export function getDisplayStopCommand(route: any) {
  const stopCommand = String(route?.localStopCommand || '').trim();
  if (stopCommand) {
    return stopCommand;
  }
  if (isDockerComposeRoute(route)) {
    return 'docker compose down';
  }
  return '';
}
