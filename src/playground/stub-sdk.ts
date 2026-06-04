import { EMPTY } from 'rxjs';
import type { FlyspaceSDK } from '@flyspace/sdk';

/**
 * Minimal SDK stub for STANDALONE dev (`ng serve` of this extension / the playground).
 * When the extension runs inside the FlySpace shell, the shell injects the real scoped SDK and
 * this stub is never used.
 */
export const stubSdk: FlyspaceSDK = {
  extensionId: 'com.example.template',
  user: {
    subject: 'dev-user',
    locale: 'en',
    profile: async () => ({ subject: 'dev-user', displayName: 'Dev User', locale: 'en' }),
    config: {
      get: async () => undefined,
      set: async () => undefined,
      watch: () => EMPTY,
    },
  },
  threads: {
    publish: async (t) => ({
      id: 'dev',
      extensionId: 'com.example.template',
      externalId: t.externalId,
      threadTypeId: t.threadTypeId,
      title: t.title,
      summary: t.summary,
      deepLink: t.deepLink,
      unreadCount: t.unreadCount ?? 0,
      isPinned: false,
      isArchived: false,
      lastActivity: new Date().toISOString(),
      createdAt: new Date().toISOString(),
    }),
    update: async () => {
      throw new Error('stub');
    },
    delete: async () => undefined,
    listOwn: async () => [],
  },
  notifications: { send: async () => undefined },
  backend: {
    request: async () => {
      throw new Error('stub: no backend in the playground');
    },
  },
  navigation: { goToCore: () => undefined, goToExtension: () => undefined },
  events: { publish: async () => undefined, observe: () => EMPTY },
  i18n: { t: (k) => k, locale: 'en' },
  ui: { toast: () => undefined, confirm: async () => true },
};
