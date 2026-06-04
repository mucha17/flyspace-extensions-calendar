import en from './en.json';
import pl from './pl.json';

const bundles: Record<string, Record<string, string>> = { en, pl };

/** Translate a UI key for the active locale, falling back to English then the raw key. */
export function translate(locale: string, key: string): string {
  return bundles[locale]?.[key] ?? bundles['en'][key] ?? key;
}
