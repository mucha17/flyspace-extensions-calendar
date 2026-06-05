import { describe, expect, it } from 'vitest';

import en from './en.json';
import pl from './pl.json';
import { translate } from './i18n';

describe('translate', () => {
  it('returns the string for the active locale', () => {
    expect(translate('pl', 'common.save')).toBe('Zapisz');
    expect(translate('en', 'common.save')).toBe('Save');
  });

  it('falls back to English for an unknown locale', () => {
    expect(translate('de', 'common.save')).toBe('Save');
  });

  it('falls back to the raw key when it exists in no bundle', () => {
    expect(translate('pl', 'does.not.exist')).toBe('does.not.exist');
    expect(translate('en', 'does.not.exist')).toBe('does.not.exist');
  });
});

describe('bundle parity', () => {
  it('pl defines exactly the same keys as the en fallback', () => {
    expect(Object.keys(pl).sort()).toEqual(Object.keys(en).sort());
  });

  it('has no empty translations', () => {
    for (const bundle of [en, pl]) {
      for (const [key, value] of Object.entries(bundle)) {
        expect(value, key).not.toBe('');
      }
    }
  });
});
