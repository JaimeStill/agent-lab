import type { ViewOptions } from './types';

export function createView(selector: string, options: ViewOptions): void {
  const init = () => {
    const element = document.querySelector(selector);
    if (!element) return;

    options.onMount();

    if (options.onDestroy) {
      window.addEventListener('beforeunload', options.onDestroy);
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
}
