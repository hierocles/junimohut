import { describe, expect, it } from "vitest";
import type { Mod } from "$lib/api/client";
import { resolveArchiveMod } from "$lib/mods/savedDownloads";
import { downloadUnknownModLabel } from "$lib/copy";

describe("resolveArchiveMod", () => {
  const installed: Mod = {
    id: "Fashion Sense::PeacefulEnd.FashionSense",
    folderPath: "Fashion Sense",
    absolutePath: "",
    manifest: {
      Name: "Fashion Sense",
      Author: "",
      Version: "1.0.0",
      Description: "",
      UniqueID: "PeacefulEnd.FashionSense",
      EntryDll: "",
      UpdateKeys: ["Nexus:9969"],
      ContentPackFor: null,
      UpdateCautionMessage: "",
      Dependencies: null,
    },
    enabled: true,
    categoryIds: [],
    updateStatus: {
      state: "current",
      latestVersion: "",
      modPageUrl: "",
      message: "",
    },
    hasConfig: false,
    hasJsonFiles: false,
    jsonFileCount: 0,
    isCoreMod: false,
    installTime: 0,
    lastUpdated: 0,
    dependencyIssues: null,
    missingDependencyCount: 0,
    containsOverwrites: false,
  };

  it("matches installed mods case-insensitively by unique id", () => {
    const resolved = resolveArchiveMod(
      {
        archivePath: "C:/downloads/mod.zip",
        uniqueId: "peacefulend.fashionsense",
        downloadedAt: 1,
      },
      [installed],
    );
    expect(resolved.displayName).toBe("Fashion Sense");
    expect(resolved.mod).toBe(installed);
  });

  it("falls back to archive mod name when not installed", () => {
    const resolved = resolveArchiveMod(
      {
        archivePath: "C:/downloads/mod.zip",
        modName: "[AT] Pet Facelift",
        uniqueId: "siamece.AT.PetFacelift",
        downloadedAt: 1,
      },
      [],
    );
    expect(resolved.displayName).toBe("[AT] Pet Facelift");
    expect(resolved.mod).toBeNull();
  });

  it("returns unknown when no metadata is available", () => {
    const resolved = resolveArchiveMod(
      { archivePath: "C:/downloads/mod.zip", downloadedAt: 1 },
      [],
    );
    expect(resolved.displayName).toBe(downloadUnknownModLabel);
  });
});
