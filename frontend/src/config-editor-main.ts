import { mount } from "svelte";
import "./app.css";
import ConfigEditorApp from "./ConfigEditorApp.svelte";
import { bootstrapDocumentTheme } from "$lib/themes/applyDocumentTheme";
import { installExternalLinks } from "./lib/wails/installExternalLinks";
import { initLocale } from "$lib/i18n";
import * as m from "$lib/paraglide/messages.js";

bootstrapDocumentTheme();
installExternalLinks();
initLocale();
document.title = m.config_editor_title_fallback();

mount(ConfigEditorApp, { target: document.getElementById("app")! });
