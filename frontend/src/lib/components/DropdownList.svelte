<script lang="ts" module>
  let nextInstanceSeq = 0;

  function nextDropdownUid(): string {
    return `dropdown-${++nextInstanceSeq}`;
  }
</script>

<script lang="ts">
  import { ChevronDown } from '@lucide/svelte';
  import { portal } from '$lib/actions/portal';
  import * as m from '$lib/paraglide/messages.js';
  import { dropdownTypeaheadJumpHint } from '$lib/i18n/helpers';

  export interface DropdownOption {
    value: string;
    label: string;
    disabled?: boolean;
  }

  interface Props {
    options: DropdownOption[];
    value?: string;
    disabled?: boolean;
    size?: 'default' | 'sm';
    id?: string;
    ariaLabel?: string;
    /** Visible field label element id (settings fields). */
    labelledById?: string;
    /** Raise above drawer stacking (settings). */
    layer?: 'default' | 'elevated';
    class?: string;
    onchange?: (value: string) => void;
  }

  interface DropdownLayoutMetrics {
    maxHeight: number;
    minWidth: number;
    gap: number;
    inset: number;
    flipAt: number;
  }

  const TYPEAHEAD_HINT_KEY = 'sdvm-dropdown-typeahead-hint-seen';

  let {
    options,
    value = $bindable(''),
    disabled = false,
    size = 'default',
    id,
    ariaLabel,
    labelledById,
    layer = 'default',
    class: className = '',
    onchange,
  }: Props = $props();

  const instanceUid = nextDropdownUid();

  let open = $state(false);
  let triggerEl = $state<HTMLButtonElement | undefined>();
  let listEl = $state<HTMLUListElement | undefined>();
  let activeIndex = $state(0);
  let typeahead = $state('');
  let showIdleHint = $state(true);
  let layoutTick = $state(0);
  let layoutMetrics = $state<DropdownLayoutMetrics>({
    maxHeight: 256,
    minWidth: 160,
    gap: 4,
    inset: 8,
    flipAt: 160,
  });
  let typeaheadTimer: ReturnType<typeof setTimeout> | undefined;
  let layoutRaf: number | undefined;

  const instanceKey = $derived(id ?? labelledById ?? instanceUid);
  const listboxId = $derived(`${instanceKey}-listbox`);
  const hintId = $derived(`${instanceKey}-hint`);

  const selectedOption = $derived(options.find((o) => o.value === value));
  const selectedLabel = $derived(selectedOption?.label ?? '');
  const triggerDisplayLabel = $derived(selectedLabel || m.dropdown_select_placeholder());

  const valueLabelId = $derived(
    labelledById ? `${labelledById}-value` : id ? `${id}-value` : undefined,
  );

  const triggerLabelledBy = $derived(
    labelledById && valueLabelId ? `${labelledById} ${valueLabelId}` : labelledById,
  );

  const accessibleName = $derived.by(() => {
    if (labelledById) return undefined;
    if (ariaLabel && selectedLabel) return `${ariaLabel}, ${selectedLabel}`;
    if (ariaLabel && !selectedLabel) return `${ariaLabel}, ${m.dropdown_select_placeholder()}`;
    return selectedLabel || ariaLabel || undefined;
  });

  const listboxDescribedBy = $derived(
    open && showIdleHint && !typeahead ? hintId : undefined,
  );

  const filteredOptions = $derived.by(() => {
    if (!typeahead) return options;
    const q = typeahead.toLowerCase();
    return options.filter((o) => o.label.toLowerCase().startsWith(q));
  });

  const activeOptionId = $derived.by(() => {
    const opt = filteredOptions[activeIndex];
    return opt ? optionDomId(instanceKey, opt.value) : undefined;
  });

  const panelStyle = $derived.by(() => {
    layoutTick;
    if (!triggerEl) return '';
    const { maxHeight, minWidth, gap, inset, flipAt } = layoutMetrics;
    const rect = triggerEl.getBoundingClientRect();
    const spaceBelow = window.innerHeight - rect.bottom - gap;
    const spaceAbove = rect.top - gap;
    const openUp = spaceBelow < Math.min(maxHeight, flipAt) && spaceAbove > spaceBelow;
    const width = Math.max(rect.width, minWidth);
    const left = Math.min(Math.max(inset, rect.left), window.innerWidth - width - inset);
    const top = openUp ? Math.max(inset, rect.top - gap) : rect.bottom + gap;
    const transform = openUp ? 'transform:translateY(-100%);' : '';
    return `left:${left}px;top:${top}px;width:${width}px;max-height:var(--dropdown-max-height);${transform}`;
  });

  function readDropdownLayout(): DropdownLayoutMetrics {
    const root = getComputedStyle(document.documentElement);
    const rem = parseFloat(root.fontSize) || 16;
    const px = (name: string, fallback: number) => {
      const raw = root.getPropertyValue(name).trim();
      if (!raw) return fallback;
      if (raw.endsWith('rem')) return parseFloat(raw) * rem;
      if (raw.endsWith('px')) return parseFloat(raw);
      return fallback;
    };
    return {
      maxHeight: px('--dropdown-max-height', 16 * rem),
      minWidth: px('--dropdown-min-width', 10 * rem),
      gap: px('--dropdown-gap', 0.25 * rem),
      inset: px('--dropdown-viewport-inset', 0.5 * rem),
      flipAt: px('--dropdown-flip-threshold', 10 * rem),
    };
  }

  function optionDomId(key: string, optionValue: string): string {
    const safe = optionValue
      .replace(/[^a-zA-Z0-9_-]/g, '-')
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, '')
      .slice(0, 48);
    return `${key}-opt-${safe || 'empty'}`;
  }

  function enabledIndices(list: DropdownOption[]): number[] {
    return list.map((o, i) => (o.disabled ? -1 : i)).filter((i) => i >= 0);
  }

  function resetTypeahead() {
    typeahead = '';
    if (typeaheadTimer) clearTimeout(typeaheadTimer);
    typeaheadTimer = undefined;
  }

  function refreshLayoutMetrics() {
    layoutMetrics = readDropdownLayout();
  }

  function scheduleLayoutBump() {
    if (layoutRaf !== undefined) return;
    layoutRaf = requestAnimationFrame(() => {
      layoutTick++;
      layoutRaf = undefined;
    });
  }

  function markHintSeen() {
    if (!showIdleHint) return;
    showIdleHint = false;
    try {
      localStorage.setItem(TYPEAHEAD_HINT_KEY, '1');
    } catch {
      /* private browsing */
    }
  }

  function dismiss(refocusTrigger = false) {
    if (!open) return;
    open = false;
    resetTypeahead();
    if (refocusTrigger) triggerEl?.focus();
  }

  function close() {
    dismiss(true);
  }

  function containsDropdownNode(node: Node | null | undefined): boolean {
    if (!node) return false;
    return !!triggerEl?.contains(node) || !!listEl?.contains(node);
  }

  function openPanel() {
    if (disabled) return;
    refreshLayoutMetrics();
    markHintSeen();
    open = true;
    resetTypeahead();
    const selectedIdx = options.findIndex((o) => o.value === value && !o.disabled);
    const firstEnabled = enabledIndices(options)[0] ?? 0;
    activeIndex = selectedIdx >= 0 ? selectedIdx : firstEnabled;
    queueMicrotask(() => {
      scrollActiveIntoView();
      listEl?.focus();
    });
  }

  function toggleOpen() {
    if (open) close();
    else openPanel();
  }

  function selectOption(opt: DropdownOption) {
    if (opt.disabled) return;
    value = opt.value;
    onchange?.(opt.value);
    close();
  }

  function scrollActiveIntoView() {
    listEl?.querySelector<HTMLElement>(`[data-option-index="${activeIndex}"]`)?.scrollIntoView({
      block: 'nearest',
    });
  }

  function moveActive(delta: number) {
    const list = filteredOptions;
    if (list.length === 0) return;
    const enabled = enabledIndices(list);
    if (enabled.length === 0) return;
    const currentPos = enabled.indexOf(activeIndex);
    let nextPos =
      currentPos < 0
        ? delta > 0
          ? 0
          : enabled.length - 1
        : (currentPos + delta + enabled.length) % enabled.length;
    activeIndex = enabled[nextPos] ?? enabled[0] ?? 0;
    scrollActiveIntoView();
  }

  function handleTypeahead(char: string) {
    if (typeaheadTimer) clearTimeout(typeaheadTimer);
    const nextBuffer = typeahead + char.toLowerCase();
    const repeat = typeahead.length === 0;

    const prefixMatches = options
      .map((o, i) => ({ o, i }))
      .filter(({ o }) => !o.disabled && o.label.toLowerCase().startsWith(nextBuffer));

    if (prefixMatches.length === 0) return;

    typeahead = nextBuffer;
    typeaheadTimer = setTimeout(resetTypeahead, 500);

    let pick = prefixMatches[0];
    if (repeat && prefixMatches.length > 1) {
      const currentValueIdx = options.findIndex((o) => o.value === value);
      const afterCurrent = prefixMatches.find(({ i }) => i > currentValueIdx);
      pick = afterCurrent ?? prefixMatches[0];
    }

    const filtered = options.filter((o) => o.label.toLowerCase().startsWith(nextBuffer));
    const filteredIdx = filtered.findIndex((o) => o.value === pick.o.value);
    activeIndex = filteredIdx >= 0 ? filteredIdx : 0;
    scrollActiveIntoView();
  }

  function onListKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
      return;
    }
    if (e.key === 'Tab') {
      dismiss(false);
      return;
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      moveActive(1);
      return;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      moveActive(-1);
      return;
    }
    if (e.key === 'Home') {
      e.preventDefault();
      const enabled = enabledIndices(filteredOptions);
      if (enabled.length) activeIndex = enabled[0];
      scrollActiveIntoView();
      return;
    }
    if (e.key === 'End') {
      e.preventDefault();
      const enabled = enabledIndices(filteredOptions);
      if (enabled.length) activeIndex = enabled[enabled.length - 1];
      scrollActiveIntoView();
      return;
    }
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      const opt = filteredOptions[activeIndex];
      if (opt) selectOption(opt);
      return;
    }
    if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault();
      handleTypeahead(e.key);
    }
  }

  $effect(() => {
    try {
      showIdleHint = localStorage.getItem(TYPEAHEAD_HINT_KEY) !== '1';
    } catch {
      showIdleHint = true;
    }
  });

  $effect(() => {
    refreshLayoutMetrics();
  });

  $effect(() => {
    if (!open) return;
    const onScrollOrResize = () => {
      scheduleLayoutBump();
    };
    const onResize = () => {
      refreshLayoutMetrics();
      scheduleLayoutBump();
    };
    window.addEventListener('scroll', onScrollOrResize);
    window.addEventListener('resize', onResize);
    return () => {
      window.removeEventListener('scroll', onScrollOrResize);
      window.removeEventListener('resize', onResize);
      if (layoutRaf !== undefined) {
        cancelAnimationFrame(layoutRaf);
        layoutRaf = undefined;
      }
    };
  });

  $effect(() => {
    if (!open || !listEl) return;
    const el = listEl;
    const onListFocusOut = () => {
      queueMicrotask(() => {
        if (!open) return;
        if (containsDropdownNode(document.activeElement)) return;
        dismiss(false);
      });
    };
    el.addEventListener('focusout', onListFocusOut);
    return () => el.removeEventListener('focusout', onListFocusOut);
  });

  $effect(() => {
    if (!open) return;
    const onPointer = (e: PointerEvent) => {
      if (e.button !== 0) return;
      const t = e.target as Node;
      if (containsDropdownNode(t)) return;
      dismiss(false);
    };
    const onFocusIn = (e: FocusEvent) => {
      if (containsDropdownNode(e.target as Node)) return;
      dismiss(false);
    };
    const frame = requestAnimationFrame(() => {
      document.addEventListener('pointerdown', onPointer);
      document.addEventListener('focusin', onFocusIn);
    });
    return () => {
      cancelAnimationFrame(frame);
      document.removeEventListener('pointerdown', onPointer);
      document.removeEventListener('focusin', onFocusIn);
    };
  });
</script>

<div class="dropdown-root {className}" class:dropdown-root--sm={size === 'sm'}>
  <button
    bind:this={triggerEl}
    {id}
    type="button"
    role="combobox"
    class="dropdown-trigger select"
    class:select-sm={size === 'sm'}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-controls={listboxId}
    aria-autocomplete="none"
    aria-labelledby={triggerLabelledBy}
    aria-label={accessibleName}
    title={selectedLabel || undefined}
    {disabled}
    onclick={toggleOpen}
    onkeydown={(e) => {
      if (disabled) return;
      if (e.key === 'Escape' && open) {
        e.preventDefault();
        close();
        return;
      }
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        if (!open) openPanel();
        else if (e.key === 'ArrowDown') moveActive(1);
        else if (e.key === 'ArrowUp') moveActive(-1);
      }
    }}
  >
    <span
      id={valueLabelId}
      class="dropdown-trigger-label truncate"
      class:type-meta={!selectedLabel}
    >
      {triggerDisplayLabel}
    </span>
    <ChevronDown size={10} strokeWidth={2.5} aria-hidden="true" class="dropdown-trigger-chevron" />
  </button>
</div>

{#if open}
  <div
    use:portal
    class="dropdown-layer"
    class:dropdown-layer--elevated={layer === 'elevated'}
    role="presentation"
  >
    <ul
      bind:this={listEl}
      id={listboxId}
      class="dropdown-panel overlay-menu-panel"
      style={panelStyle}
      role="listbox"
      tabindex="-1"
      aria-activedescendant={activeOptionId}
      aria-labelledby={triggerLabelledBy}
      aria-describedby={listboxDescribedBy}
      aria-label={accessibleName}
      onkeydown={onListKeydown}
    >
      {#if typeahead}
        <li class="overlay-menu-hint overlay-menu-hint--leading type-caption type-meta" role="presentation">
          <span aria-hidden="true">{dropdownTypeaheadJumpHint(typeahead)}</span>
        </li>
      {:else if showIdleHint}
        <li
          id={hintId}
          class="overlay-menu-hint overlay-menu-hint--leading type-caption type-meta"
          role="presentation"
        >
          {m.dropdown_typeahead_idle_hint()}
        </li>
      {/if}
      {#if filteredOptions.length === 0}
        <li class="dropdown-empty type-caption type-meta" role="presentation">{m.dropdown_empty_options()}</li>
      {:else}
        {#each filteredOptions as opt, i (`${instanceKey}-${opt.value}`)}
          <li role="presentation">
            <button
              id={optionDomId(instanceKey, opt.value)}
              type="button"
              class="overlay-menu-item"
              class:overlay-menu-item--active={i === activeIndex}
              role="option"
              aria-selected={value === opt.value}
              disabled={opt.disabled}
              data-option-index={i}
              onclick={() => selectOption(opt)}
              onmouseenter={() => (activeIndex = i)}
            >
              <span class="overlay-menu-check" aria-hidden="true">{value === opt.value ? '✓' : ''}</span>
              <span class="truncate">{opt.label}</span>
            </button>
          </li>
        {/each}
      {/if}
    </ul>
  </div>
{/if}

<style>
  .dropdown-root {
    position: relative;
    display: block;
    min-width: 0;
  }

  .dropdown-root--sm {
    min-width: 5rem;
    max-width: 11rem;
  }

  .dropdown-trigger {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    width: 100%;
    padding-inline: var(--space-3) var(--space-2);
    text-align: left;
    cursor: pointer;
    background-image: none;
  }

  .dropdown-trigger.select-sm {
    padding-inline: var(--space-2) var(--space-2);
  }

  .dropdown-trigger-label {
    flex: 1;
    min-width: 0;
  }

  :global(.dropdown-trigger-chevron) {
    flex-shrink: 0;
    color: var(--color-surface-400);
    transition:
      transform var(--motion-fast) var(--ease-out-quart),
      color var(--motion-fast) var(--ease-out-quart);
  }

  .dropdown-trigger[aria-expanded='true'] :global(.dropdown-trigger-chevron) {
    transform: rotate(180deg);
  }

  .dropdown-trigger:hover:not(:disabled) :global(.dropdown-trigger-chevron),
  .dropdown-trigger:focus-visible :global(.dropdown-trigger-chevron) {
    color: var(--color-surface-300);
  }

  .dropdown-layer {
    position: fixed;
    inset: 0;
    z-index: var(--z-dropdown);
    pointer-events: none;
  }

  .dropdown-layer--elevated {
    z-index: calc(var(--z-drawer) + 1);
  }

  .dropdown-panel {
    position: fixed;
    pointer-events: auto;
    margin: 0;
    padding-block: var(--space-1);
    list-style: none;
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .dropdown-panel :global(.overlay-menu-item) {
    min-height: var(--dropdown-item-min-h);
  }

  .dropdown-empty {
    margin: 0;
    padding: var(--space-2) var(--space-3);
  }

  .dropdown-panel :global(.overlay-menu-item--active) {
    background-color: color-mix(in oklab, var(--color-surface-800) 60%, transparent);
  }

  @media (prefers-reduced-motion: reduce) {
    .dropdown-trigger {
      transition: none;
    }

    :global(.dropdown-trigger-chevron) {
      transition: none;
    }
  }
</style>
