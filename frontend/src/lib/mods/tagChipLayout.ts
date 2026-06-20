import type { Category } from "$lib/api/client";

/** Approximate chip metrics at `--type-meta` semibold (matches `.tag-chip` in ModGrid). */
const CHIP_H_PAD = 16;
const CHIP_GAP = 4;
const CHAR_WIDTH = 6.5;
const OVERFLOW_CHIP_WIDTH = 36;
const CELL_H_PAD = 16;

export type TagChipLayout = {
  visible: Category[];
  overflowCount: number;
  overflowLabel: string;
};

export function layoutTagChips(
  categories: Category[],
  columnWidthPx: number,
): TagChipLayout {
  if (categories.length === 0) {
    return { visible: [], overflowCount: 0, overflowLabel: "" };
  }

  const budget = Math.max(0, columnWidthPx - CELL_H_PAD);
  const visible: Category[] = [];
  let used = 0;

  for (let i = 0; i < categories.length; i++) {
    const cat = categories[i];
    const chipW = CHIP_H_PAD + cat.name.length * CHAR_WIDTH;
    const hiddenAfter = categories.length - i - 1;
    const reserve = hiddenAfter > 0 ? OVERFLOW_CHIP_WIDTH + CHIP_GAP : 0;

    if (visible.length > 0 && used + chipW + reserve > budget) {
      break;
    }

    visible.push(cat);
    used += chipW + CHIP_GAP;

    if (visible.length === 1 && hiddenAfter > 0 && used + reserve > budget) {
      break;
    }
  }

  const overflowCount = categories.length - visible.length;
  const overflowLabel =
    overflowCount > 0
      ? categories
          .slice(visible.length)
          .map((c) => c.name)
          .join(", ")
      : "";

  return { visible, overflowCount, overflowLabel };
}
