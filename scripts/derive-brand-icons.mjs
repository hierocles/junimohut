import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const sourcePath = path.join(root, "assets/icon.svg");
const svg = fs.readFileSync(sourcePath, "utf8");
const inner = svg.replace(/^<svg[^>]*>/, "").replace(/<\/svg>\s*$/, "");

const crop = "88 72 238 268";
const bg = "#152820";
const bgInner = "#1a3228";

const croppedMark = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${crop}" width="24" height="24" fill-rule="evenodd" clip-rule="evenodd" aria-hidden="true">${inner}</svg>`;

const favicon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32" aria-hidden="true">
  <rect width="32" height="32" rx="7" fill="${bg}"/>
  <svg x="2" y="2" width="28" height="28" viewBox="${crop}" fill-rule="evenodd" clip-rule="evenodd">${inner}</svg>
</svg>`;

const appIcon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024" aria-hidden="true">
  <rect width="1024" height="1024" rx="224" fill="${bg}"/>
  <rect x="96" y="96" width="832" height="832" rx="192" fill="${bgInner}" opacity="0.45"/>
  <svg x="64" y="64" width="896" height="896" viewBox="${crop}" fill-rule="evenodd" clip-rule="evenodd">${inner}</svg>
</svg>`;

const brandDir = path.join(root, "frontend/src/lib/assets/brand");
const buildDir = path.join(root, "build");
const appIconSvgPath = path.join(brandDir, "app-icon.svg");
const appiconPngPath = path.join(buildDir, "appicon.png");

fs.writeFileSync(path.join(brandDir, "junimo-hut-mark.svg"), croppedMark);
fs.writeFileSync(path.join(brandDir, "brand-mark.svg"), croppedMark);
fs.writeFileSync(path.join(brandDir, "brand-mark-mono.svg"), croppedMark);
fs.writeFileSync(path.join(brandDir, "favicon.svg"), favicon);
fs.writeFileSync(path.join(root, "frontend/public/favicon.svg"), favicon);
fs.writeFileSync(appIconSvgPath, appIcon);

fs.mkdirSync(buildDir, { recursive: true });

execSync(
  `npx --yes @resvg/resvg-js-cli --shape-rendering 1 "${appIconSvgPath}" "${appiconPngPath}"`,
  { stdio: "inherit", cwd: root, shell: true },
);

console.log("Derived brand icons from", sourcePath);
console.log("Wrote", appiconPngPath);

try {
  execSync(
    "wails3 generate icons -input appicon.png -macfilename darwin/icons.icns -windowsfilename windows/icon.ico -iconcomposerinput appicon.icon -macassetdir darwin",
    { stdio: "inherit", cwd: buildDir, shell: true },
  );
  console.log("Regenerated build/windows/icon.ico and build/darwin/icons.icns");
} catch {
  console.warn(
    "Skipped wails3 icon bundle generation. Install the Wails CLI, then run: wails3 task generate:icons",
  );
}
