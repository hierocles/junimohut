const OPTIONAL_THEMES = new Set(["cerberus", "mona", "vox", "stardew-light"]);

const THEME_STORAGE_KEY = "sdvm-theme";

let optionalThemesLoaded = false;
let optionalThemesLoading: Promise<void> | null = null;

async function ensureOptionalThemesLoaded(): Promise<void> {
  if (optionalThemesLoaded) return;
  if (!optionalThemesLoading) {
    optionalThemesLoading = import("../../themes/optional.css").then(() => {
      optionalThemesLoaded = true;
    });
  }
  await optionalThemesLoading;
}

/** Restore last theme attribute before settings load (CSS loads later). */
export function bootstrapDocumentTheme(): void {
  try {
    const cached = localStorage.getItem(THEME_STORAGE_KEY);
    document.documentElement.setAttribute(
      "data-theme",
      cached ?? "stardew-dark",
    );
    return;
  } catch {
    /* storage unavailable */
  }
  document.documentElement.setAttribute("data-theme", "stardew-dark");
}

function scheduleOptionalThemesLoad(): void {
  const run = () => void ensureOptionalThemesLoaded();
  if (typeof requestIdleCallback === "function") {
    requestIdleCallback(run);
  } else {
    setTimeout(run, 0);
  }
}

/** Apply a theme attribute and lazy-load alternate theme CSS when needed. */
export function applyDocumentTheme(
  theme: string,
  options?: { persist?: boolean },
): void {
  document.documentElement.setAttribute("data-theme", theme);
  if (options?.persist !== false) {
    try {
      localStorage.setItem(THEME_STORAGE_KEY, theme);
    } catch {
      /* storage unavailable */
    }
  }
  if (OPTIONAL_THEMES.has(theme)) {
    scheduleOptionalThemesLoad();
  }
}
