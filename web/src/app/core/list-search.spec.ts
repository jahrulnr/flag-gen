import { effectiveSearchQuery, SEARCH_MIN_CHARS } from './list-search';

describe('effectiveSearchQuery', () => {
  it('returns empty for whitespace-only input', () => {
    expect(effectiveSearchQuery('   ')).toBe('');
  });

  it('returns empty when shorter than minimum length', () => {
    expect(effectiveSearchQuery('a'.repeat(SEARCH_MIN_CHARS - 1))).toBe('');
  });

  it('returns trimmed query at minimum length', () => {
    expect(effectiveSearchQuery('  abc  ')).toBe('abc');
  });

  it('returns longer trimmed queries', () => {
    expect(effectiveSearchQuery('  qa_ui_901  ')).toBe('qa_ui_901');
  });
});
