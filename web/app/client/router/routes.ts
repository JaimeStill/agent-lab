import type { RouteConfig } from './types';

export const routes: Record<string, RouteConfig> = {
  '': { component: 'lab-home-view', title: 'Home' },
  '*': { component: 'lab-not-found-view', title: 'Not Found' },
};
