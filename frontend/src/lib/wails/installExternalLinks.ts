import { Browser } from '@wailsio/runtime';

import { isWailsHost } from './windowApi';

function isHttpUrl(url: string): boolean {
  return /^https?:\/\//i.test(url);
}

/** Route external URLs to the OS default browser when running inside Wails. */
export function installExternalLinks(): void {
  if (!isWailsHost()) return;

  const nativeOpen = window.open.bind(window);
  window.open = (url?: string | URL, target?: string, features?: string): WindowProxy | null => {
    if (typeof url === 'string' && isHttpUrl(url)) {
      void Browser.OpenURL(url);
      return null;
    }
    if (url instanceof URL && isHttpUrl(url.toString())) {
      void Browser.OpenURL(url);
      return null;
    }
    return nativeOpen(url, target, features);
  };

  document.addEventListener(
    'click',
    (event) => {
      const anchor = (event.target as Element | null)?.closest('a[href]');
      if (!anchor) return;

      const href = anchor.getAttribute('href');
      if (!href || !isHttpUrl(href)) return;

      event.preventDefault();
      void Browser.OpenURL(href);
    },
    true,
  );
}
