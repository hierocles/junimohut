import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import { bootstrapDocumentTheme } from "$lib/themes/applyDocumentTheme";
import { installExternalLinks } from "./lib/wails/installExternalLinks";
import { initLocale } from "$lib/i18n";
import * as m from "$lib/paraglide/messages.js";

bootstrapDocumentTheme();
installExternalLinks();
initLocale();
document.title = m.brand_wordmark_title();

mount(App, { target: document.getElementById("app")! });
