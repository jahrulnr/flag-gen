/** Human-readable timestamps for audit / history tables; falls back gracefully. */
export function formatDisplayTimestamp(raw: unknown): string {
  if (raw == null || raw === '') {
    return '—';
  }
  const s = String(raw);
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) {
    return s;
  }
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(d);
}
