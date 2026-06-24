import { getLocale, locales, setLocale } from "$lib/paraglide/runtime.js";

type Locale = (typeof locales)[number];

function resolveLocale(preferred?: string): Locale {
  if (preferred && locales.includes(preferred as Locale)) {
    return preferred as Locale;
  }
  return getLocale();
}

export function initLocale(preferred?: string): string {
  const tag = resolveLocale(preferred);
  setLocale(tag, { reload: false });
  document.documentElement.lang = tag;
  return tag;
}

export function applyLocale(tag: string): void {
  if (tag === getLocale()) return;
  if (locales.includes(tag as Locale)) {
    setLocale(tag as Locale);
  }
}
