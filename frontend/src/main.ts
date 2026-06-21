import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import { bootstrapDocumentTheme } from "$lib/themes/applyDocumentTheme";
import { installExternalLinks } from "./lib/wails/installExternalLinks";

bootstrapDocumentTheme();
installExternalLinks();

mount(App, { target: document.getElementById("app")! });
