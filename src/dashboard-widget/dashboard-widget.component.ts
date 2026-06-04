import { Component, Input, OnInit, inject } from '@angular/core';
import { FLYSPACE_SDK } from '@flyspace/sdk';

import { CalendarApi } from '../data/calendar-api';
import type { CalendarEvent } from '../data/event';
import { addDays } from '../data/date-utils';
import { translate } from '../i18n/i18n';

/** Dashboard widget: the user's next few upcoming events. Provides its own CalendarApi because the
 *  shell mounts it with its own scoped injector. */
@Component({
  selector: 'fly-calendar-widget',
  standalone: true,
  providers: [CalendarApi],
  template: `
    <section class="w">
      <header class="w-head">{{ t('widget.title') }}</header>
      @if (loading) {
        <p class="w-muted">…</p>
      } @else if (upcoming.length === 0) {
        <p class="w-muted">{{ t('widget.empty') }}</p>
      } @else {
        <ul class="w-list">
          @for (e of upcoming; track e.id) {
            <li class="w-item">
              <span class="w-when">{{ when(e) }}</span>
              <span class="w-title">{{ e.title }}</span>
            </li>
          }
        </ul>
      }
    </section>
  `,
  styles: [
    `
      .w {
        padding: var(--fly-space-md);
        background: var(--fly-color-surface);
        color: var(--fly-color-on-surface);
        border-radius: var(--fly-radius-lg);
        box-shadow: var(--fly-shadow-sm);
        font-family: var(--fly-font-family-base);
      }
      .w-head {
        font-size: var(--fly-font-size-md);
        font-weight: var(--fly-font-weight-bold);
        margin-bottom: var(--fly-space-sm);
      }
      .w-muted {
        margin: 0;
        opacity: 0.6;
        font-size: var(--fly-font-size-sm);
      }
      .w-list {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: var(--fly-space-xs);
      }
      .w-item {
        display: flex;
        gap: var(--fly-space-sm);
        font-size: var(--fly-font-size-sm);
      }
      .w-when {
        flex: none;
        min-width: 9ch;
        color: var(--fly-color-primary);
        font-weight: var(--fly-font-weight-medium);
      }
      .w-title {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    `,
  ],
})
export class DashboardWidgetComponent implements OnInit {
  @Input() size: 'small' | 'medium' | 'large' = 'medium';

  private readonly api = inject(CalendarApi);
  private readonly sdk = inject(FLYSPACE_SDK);

  upcoming: CalendarEvent[] = [];
  loading = true;

  async ngOnInit(): Promise<void> {
    const now = new Date();
    try {
      const events = await this.api.list(now.toISOString(), addDays(now, 30).toISOString());
      this.upcoming = events
        .filter((e) => new Date(e.end).getTime() >= now.getTime())
        .sort((a, b) => new Date(a.start).getTime() - new Date(b.start).getTime())
        .slice(0, this.size === 'small' ? 3 : 5);
    } catch {
      this.upcoming = [];
    }
    this.loading = false;
  }

  when(e: CalendarEvent): string {
    const opts: Intl.DateTimeFormatOptions = e.allDay
      ? { month: 'short', day: 'numeric' }
      : { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' };
    return new Intl.DateTimeFormat(this.sdk.i18n.locale, opts).format(new Date(e.start));
  }

  t(key: string): string {
    return translate(this.sdk.i18n.locale, key);
  }
}
