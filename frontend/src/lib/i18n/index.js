import { GetTranslations } from "$lib/api";
const locales = {
    en: {},
};
export function t(key, locale = "en", fallback = key) {
    return locales[locale]?.[key] ?? fallback;
}
export async function loadTranslations(locale) {
    try {
        const tr = (await GetTranslations(locale)) ?? {};
        locales[locale] = tr;
        return tr;
    }
    catch {
        return {};
    }
}
//# sourceMappingURL=index.js.map