import { GetTranslations } from "$lib/api";

const locales: Record<string, Record<string, string | undefined>> = {
  en: {},
};

export function t(key: string, locale = "en", fallback = key): string {
  return locales[locale]?.[key] ?? fallback;
}

export async function loadTranslations(
  locale: string,
): Promise<Record<string, string | undefined>> {
  try {
    const tr = (await GetTranslations(locale)) ?? {};
    locales[locale] = tr;
    return tr;
  } catch {
    return {};
  }
}
