import { describe, expect, it } from "vitest";
import type { Category } from "$lib/api/client";
import { layoutTagChips } from "./tagChipLayout";

function cat(id: string, name: string): Category {
  return {
    id,
    name,
    color: "#6366f1",
    visible: true,
    sortOrder: 0,
  };
}

describe("layoutTagChips", () => {
  it("returns empty layout for no categories", () => {
    expect(layoutTagChips([], 160)).toEqual({
      visible: [],
      overflowCount: 0,
      overflowLabel: "",
    });
  });

  it("shows all tags when they fit", () => {
    const cats = [cat("1", "QoL"), cat("2", "UI")];
    const result = layoutTagChips(cats, 200);
    expect(result.visible).toHaveLength(2);
    expect(result.overflowCount).toBe(0);
  });

  it("reserves +N when tags overflow", () => {
    const cats = [
      cat("1", "Quality of life"),
      cat("2", "User interface"),
      cat("3", "Cheats"),
    ];
    const result = layoutTagChips(cats, 120);
    expect(result.visible.length).toBeGreaterThan(0);
    expect(result.overflowCount).toBeGreaterThan(0);
    expect(result.overflowLabel).toContain("Cheats");
  });
});
