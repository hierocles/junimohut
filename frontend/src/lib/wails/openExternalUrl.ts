import { Browser } from '@wailsio/runtime';

import { isWailsHost } from './windowApi';

export async function openExternalUrl(url: string | URL): Promise<void> {
  const href = url.toString();
  if (!href) return;

  if (isWailsHost()) {
    await Browser.OpenURL(href);
    return;
  }

  window.open(href, '_blank', 'noopener,noreferrer');
}
