import { Component, OnInit, inject } from '@angular/core';
import { FLYSPACE_SDK } from '@flyspace/sdk';

import { translate } from '../i18n/i18n';

/** SettingsPanel: per-user calendar preferences, stored via sdk.user.config (core-owned, cross-tab
 *  synced). Currently the week-start day, which MainView reads. */
@Component({
  selector: 'fly-calendar-settings',
  standalone: true,
  template: `
    <section class="s">
      <label class="s-field">
        <span>{{ t('settings.weekStart') }}</span>
        <select [value]="weekStart" (change)="setWeekStart($event)">
          <option [value]="1">{{ t('settings.monday') }}</option>
          <option [value]="0">{{ t('settings.sunday') }}</option>
        </select>
      </label>
    </section>
  `,
  styles: [
    `
      .s {
        padding: var(--fly-space-md);
        color: var(--fly-color-on-surface);
        font-family: var(--fly-font-family-base);
      }
      .s-field {
        display: flex;
        flex-direction: column;
        gap: var(--fly-space-xs);
        font-size: var(--fly-font-size-sm);
      }
      select {
        font: inherit;
        color: inherit;
        padding: var(--fly-space-sm);
        background: var(--fly-color-background);
        border: var(--fly-border-width-thin) solid var(--fly-color-border);
        border-radius: var(--fly-radius-sm);
      }
    `,
  ],
})
export class SettingsPanelComponent implements OnInit {
  private readonly sdk = inject(FLYSPACE_SDK);

  weekStart = 1;

  async ngOnInit(): Promise<void> {
    try {
      const stored = await this.sdk.user.config.get<number>('weekStart');
      this.weekStart = stored === 0 ? 0 : 1;
    } catch {
      this.weekStart = 1;
    }
  }

  async setWeekStart(e: Event): Promise<void> {
    const value = Number((e.target as HTMLSelectElement).value) === 0 ? 0 : 1;
    this.weekStart = value;
    try {
      await this.sdk.user.config.set('weekStart', value);
      this.sdk.ui.toast({ en: 'Saved.', pl: 'Zapisano.' }, 'success');
    } catch {
      this.sdk.ui.toast({ en: 'Could not save.', pl: 'Nie udało się zapisać.' }, 'error');
    }
  }

  t(key: string): string {
    return translate(this.sdk.i18n.locale, key);
  }
}
