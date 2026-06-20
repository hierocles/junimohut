import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import { installExternalLinks } from "./lib/wails/installExternalLinks";

installExternalLinks();

mount(App, { target: document.getElementById("app")! });
