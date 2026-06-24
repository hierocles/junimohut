import { getLocale } from "$lib/paraglide/runtime.js";

export function appLocale(): string {
  return getLocale();
}
