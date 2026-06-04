import { Component, OnInit, inject } from '@angular/core';
import { FLYSPACE_SDK } from '@flyspace/sdk';

import { CalendarApi } from '../data/calendar-api';
import type { CalendarEvent, EventInput } from '../data/event';
import { addDays, addMonths, monthGrid, occursOn, sameDay, startOfDay } from '../data/date-utils';
import { translate } from '../i18n/i18n';
import { EventDialogComponent } from './event-dialog.component';

/** MainView — the month calendar. Loads the visible month's events from the backend and hosts the
 *  create/edit dialog. */
@Component({
  selector: 'fly-calendar-main',
  standalone: true,
  imports: [EventDialogComponent],
  template: `
    <section class="cal">
      <header class="cal-head">
        <div class="cal-nav">
          <button type="button" (click)="shift(-1)" [attr.aria-label]="t('cal.prev')">‹</button>
          <h1>{{ monthLabel }}</h1>
          <button type="button" (click)="shift(1)" [attr.aria-label]="t('cal.next')">›</button>
        </div>
        <div class="cal-head-actions">
          <button type="button" (click)="goToday()">{{ t('cal.today') }}</button>
          <button type="button" class="cal-primary" (click)="openNew(today())">{{ t('cal.new') }}</button>
        </div>
      </header>

      @if (loadError) {
        <p class="cal-error">{{ t('cal.loadError') }}</p>
      }

      <div class="cal-weekdays">
        @for (w of weekdays; track w) {
          <div>{{ w }}</div>
        }
      </div>

      <div class="cal-grid">
        @for (day of grid; track day.getTime()) {
          <div
            class="cal-cell"
            role="button"
            tabindex="0"
            [class.cal-other]="!inMonth(day)"
            [class.cal-today]="isToday(day)"
            (click)="openNew(day)"
            (keydown.enter)="openNew(day)"
          >
            <span class="cal-daynum">{{ day.getDate() }}</span>
            @for (e of eventsOn(day); track e.id) {
              <button type="button" class="cal-chip" [class.cal-allday]="e.allDay" (click)="openEdit(e, $event)">
                {{ e.title }}
              </button>
            }
          </div>
        }
      </div>

      @if (dialogOpen) {
        <fly-calendar-event-dialog
          [event]="editing"
          [defaultDate]="selectedDate"
          (save)="onSave($event)"
          (delete)="onDelete($event)"
          (dismiss)="closeDialog()"
        />
      }
    </section>
  `,
  styles: [
    `
      .cal {
        padding: var(--fly-space-md);
        color: var(--fly-color-on-background);
        font-family: var(--fly-font-family-base);
      }
      .cal-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: var(--fly-space-md);
        margin-bottom: var(--fly-space-md);
      }
      .cal-nav {
        display: flex;
        align-items: center;
        gap: var(--fly-space-sm);
      }
      h1 {
        margin: 0;
        min-width: 12ch;
        text-align: center;
        font-size: var(--fly-font-size-lg);
        font-weight: var(--fly-font-weight-bold);
      }
      .cal-head-actions {
        display: flex;
        gap: var(--fly-space-sm);
      }
      button {
        font: inherit;
        cursor: pointer;
        padding: var(--fly-space-xs) var(--fly-space-sm);
        border: var(--fly-border-width-thin) solid var(--fly-color-border);
        border-radius: var(--fly-radius-sm);
        background: var(--fly-color-surface);
        color: var(--fly-color-on-surface);
      }
      .cal-primary {
        background: var(--fly-color-primary);
        color: var(--fly-color-on-primary);
        border-color: transparent;
      }
      .cal-weekdays {
        display: grid;
        grid-template-columns: repeat(7, 1fr);
        gap: var(--fly-space-xs);
        font-size: var(--fly-font-size-xs);
        font-weight: var(--fly-font-weight-medium);
        color: var(--fly-color-on-surface);
        opacity: 0.7;
      }
      .cal-weekdays > div {
        padding: var(--fly-space-xs);
        text-align: center;
      }
      .cal-grid {
        display: grid;
        grid-template-columns: repeat(7, 1fr);
        gap: var(--fly-space-xs);
        margin-top: var(--fly-space-xs);
      }
      .cal-cell {
        display: flex;
        flex-direction: column;
        gap: var(--fly-space-xs);
        min-height: 5rem;
        padding: var(--fly-space-xs);
        background: var(--fly-color-surface);
        border: var(--fly-border-width-thin) solid var(--fly-color-border);
        border-radius: var(--fly-radius-sm);
        cursor: pointer;
      }
      .cal-other {
        opacity: 0.45;
      }
      .cal-today {
        border-color: var(--fly-color-primary);
      }
      .cal-daynum {
        font-size: var(--fly-font-size-xs);
        font-weight: var(--fly-font-weight-medium);
      }
      .cal-chip {
        text-align: left;
        padding: 0 var(--fly-space-xs);
        border: none;
        border-radius: var(--fly-radius-sm);
        background: var(--fly-color-surface-muted);
        color: var(--fly-color-on-surface);
        font-size: var(--fly-font-size-xs);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
      .cal-allday {
        background: var(--fly-color-primary);
        color: var(--fly-color-on-primary);
      }
      .cal-error {
        color: var(--fly-color-danger);
        font-size: var(--fly-font-size-sm);
      }
    `,
  ],
})
export class MainViewComponent implements OnInit {
  private readonly api = inject(CalendarApi);
  private readonly sdk = inject(FLYSPACE_SDK);

  month = new Date(new Date().getFullYear(), new Date().getMonth(), 1);
  weekStart: 0 | 1 = 1;
  events: CalendarEvent[] = [];
  grid: Date[] = [];
  weekdays: string[] = [];
  monthLabel = '';
  loadError = false;

  dialogOpen = false;
  editing: CalendarEvent | null = null;
  selectedDate = new Date();

  async ngOnInit(): Promise<void> {
    const stored = await this.safeConfig();
    this.weekStart = stored === 0 ? 0 : 1;
    await this.refresh();
  }

  today(): Date {
    return new Date();
  }

  async shift(months: number): Promise<void> {
    this.month = addMonths(this.month, months);
    await this.refresh();
  }

  async goToday(): Promise<void> {
    const now = new Date();
    this.month = new Date(now.getFullYear(), now.getMonth(), 1);
    await this.refresh();
  }

  inMonth(day: Date): boolean {
    return day.getMonth() === this.month.getMonth();
  }

  isToday(day: Date): boolean {
    return sameDay(day, new Date());
  }

  eventsOn(day: Date): CalendarEvent[] {
    return this.events.filter((e) => occursOn(e, day));
  }

  openNew(day: Date): void {
    this.editing = null;
    this.selectedDate = day;
    this.dialogOpen = true;
  }

  openEdit(event: CalendarEvent, e: Event): void {
    e.stopPropagation();
    this.editing = event;
    this.dialogOpen = true;
  }

  closeDialog(): void {
    this.dialogOpen = false;
    this.editing = null;
  }

  async onSave(input: EventInput): Promise<void> {
    try {
      if (this.editing) {
        await this.api.replace(this.editing.id, input);
      } else {
        await this.api.create(input);
      }
      this.closeDialog();
      await this.refresh();
    } catch {
      this.sdk.ui.toast({ en: 'Could not save the event.', pl: 'Nie udało się zapisać wydarzenia.' }, 'error');
    }
  }

  async onDelete(id: string): Promise<void> {
    try {
      await this.api.remove(id);
      this.closeDialog();
      await this.refresh();
    } catch {
      this.sdk.ui.toast({ en: 'Could not delete the event.', pl: 'Nie udało się usunąć wydarzenia.' }, 'error');
    }
  }

  t(key: string): string {
    return translate(this.sdk.i18n.locale, key);
  }

  private async refresh(): Promise<void> {
    this.grid = monthGrid(this.month, this.weekStart);
    this.weekdays = this.buildWeekdays();
    this.monthLabel = new Intl.DateTimeFormat(this.sdk.i18n.locale, { month: 'long', year: 'numeric' }).format(this.month);
    const from = startOfDay(this.grid[0]).toISOString();
    const to = addDays(startOfDay(this.grid[this.grid.length - 1]), 1).toISOString();
    try {
      this.events = await this.api.list(from, to);
      this.loadError = false;
    } catch {
      this.events = [];
      this.loadError = true;
    }
  }

  private buildWeekdays(): string[] {
    const fmt = new Intl.DateTimeFormat(this.sdk.i18n.locale, { weekday: 'short' });
    const knownSunday = new Date(2023, 0, 1); // 2023-01-01 was a Sunday
    return Array.from({ length: 7 }, (_, i) => fmt.format(addDays(knownSunday, (this.weekStart + i) % 7)));
  }

  private async safeConfig(): Promise<number | undefined> {
    try {
      return await this.sdk.user.config.get<number>('weekStart');
    } catch {
      return undefined;
    }
  }
}
