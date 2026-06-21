import { mount } from "svelte";
import "./app.css";
import ConfigEditorApp from "./ConfigEditorApp.svelte";
import { bootstrapDocumentTheme } from "$lib/themes/applyDocumentTheme";
import { installExternalLinks } from "./lib/wails/installExternalLinks";

bootstrapDocumentTheme();
installExternalLinks();

mount(ConfigEditorApp, { target: document.getElementById("app")! });
