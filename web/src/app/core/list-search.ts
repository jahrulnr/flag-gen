export const SEARCH_DEBOUNCE_MS = 250;
export const SEARCH_MIN_CHARS = 3;

/** Trimmed query sent to list APIs; shorter than min length returns empty (full list). */
export function effectiveSearchQuery(raw: string): string {
  const t = raw.trim();
  if (t.length > 0 && t.length < SEARCH_MIN_CHARS) {
    return '';
  }
  return t;
}

export type SearchDebounceHandle = ReturnType<typeof setTimeout> | null;

export function scheduleDebouncedSearch(
  handle: SearchDebounceHandle,
  raw: string,
  onRun: () => void,
): SearchDebounceHandle {
  if (handle) {
    clearTimeout(handle);
  }
  return setTimeout(() => {
    onRun();
  }, SEARCH_DEBOUNCE_MS);
}
