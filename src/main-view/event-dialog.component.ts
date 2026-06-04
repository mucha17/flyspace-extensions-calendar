import { Component, EventEmitter, Input, OnChanges, Output, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { FLYSPACE_SDK } from '@flyspace/sdk';

import type { CalendarEvent, EventInput } from '../data/event';
import {
  dateInputToISO,
  dateTimeInputToISO,
  toDateInput,
  toDateTimeInput,
} from '../data/date-utils';
import { translate } from '../i18n/i18n';

/** Modal form for creating, editing, or deleting an event. */
@Component({
  selector: 'fly-calendar-event-dialog',
  standalone: true,
  imports: [FormsModule],
  template: `
    <div
      class="cal-backdrop"
      role="button"
      tabindex="0"
      [attr.aria-label]="t('common.cancel')"
      (click)="dismiss.emit()"
      (keydown.escape)="dismiss.emit()"
      (keydown.enter)="dismiss.emit()"
    ></div>
    <form class="cal-dialog" (submit)="onSave($event)">
      <h2>{{ event ? t('event.edit') : t('event.new') }}</h2>

      <label class="cal-field">
        <span>{{ t('event.title') }}</span>
        <input name="title" [(ngModel)]="title" autocomplete="off" />
      </label>

      <label class="cal-check">
        <input type="checkbox" name="allDay" [(ngModel)]="allDay" (ngModelChange)="onAllDayToggle()" />
        <span>{{ t('event.allDay') }}</span>
      </label>

      <div class="cal-times">
        <label class="cal-field">
          <span>{{ t('event.start') }}</span>
          <input [type]="allDay ? 'date' : 'datetime-local'" name="start" [(ngModel)]="startInput" />
        </label>
        <label class="cal-field">
          <span>{{ t('event.end') }}</span>
          <input [type]="allDay ? 'date' : 'datetime-local'" name="end" [(ngModel)]="endInput" />
        </label>
      </div>

      <label class="cal-field">
        <span>{{ t('event.notes') }}</span>
        <textarea name="notes" [(ngModel)]="notes" rows="3"></textarea>
      </label>

      @if (error) {
        <p class="cal-error">{{ error }}</p>
      }

      <div class="cal-actions">
        @if (event) {
          <button type="button" class="cal-danger" (click)="delete.emit(event.id)">{{ t('event.delete') }}</button>
        }
        <span class="cal-spacer"></span>
        <button type="button" (click)="dismiss.emit()">{{ t('common.cancel') }}</button>
        <button type="submit" class="cal-primary">{{ t('common.save') }}</button>
      </div>
    </form>
  `,
  styles: [
    `
      :host {
        position: fixed;
        inset: 0;
        z-index: var(--fly-z-index-modal);
        display: grid;
        place-items: center;
      }
      .cal-backdrop {
        position: absolute;
        inset: 0;
        background: var(--fly-color-on-surface);
        opacity: 0.4;
      }
      .cal-dialog {
        position: relative;
        width: min(90vw, 28rem);
        display: flex;
        flex-direction: column;
        gap: var(--fly-space-md);
        padding: var(--fly-space-lg);
        background: var(--fly-color-surface);
        color: var(--fly-color-on-surface);
        border-radius: var(--fly-radius-lg);
        box-shadow: var(--fly-shadow-lg);
      }
      h2 {
        margin: 0;
        font-size: var(--fly-font-size-lg);
        font-weight: var(--fly-font-weight-bold);
      }
      .cal-field {
        display: flex;
        flex-direction: column;
        gap: var(--fly-space-xs);
        font-size: var(--fly-font-size-sm);
      }
      input,
      textarea {
        font: inherit;
        color: inherit;
        padding: var(--fly-space-sm);
        background: var(--fly-color-background);
        border: var(--fly-border-width-thin) solid var(--fly-color-border);
        border-radius: var(--fly-radius-sm);
      }
      .cal-check {
        display: flex;
        align-items: center;
        gap: var(--fly-space-sm);
        font-size: var(--fly-font-size-sm);
      }
      .cal-times {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: var(--fly-space-md);
      }
      .cal-error {
        margin: 0;
        color: var(--fly-color-danger);
        font-size: var(--fly-font-size-sm);
      }
      .cal-actions {
        display: flex;
        align-items: center;
        gap: var(--fly-space-sm);
      }
      .cal-spacer {
        flex: 1;
      }
      button {
        font: inherit;
        cursor: pointer;
        padding: var(--fly-space-sm) var(--fly-space-md);
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
      .cal-danger {
        color: var(--fly-color-danger);
      }
    `,
  ],
})
export class EventDialogComponent implements OnChanges {
  @Input() event: CalendarEvent | null = null;
  @Input() defaultDate: Date = new Date();
  @Output() save = new EventEmitter<EventInput>();
  @Output() delete = new EventEmitter<string>();
  @Output() dismiss = new EventEmitter<void>();

  private readonly sdk = inject(FLYSPACE_SDK);

  title = '';
  allDay = false;
  notes = '';
  startInput = '';
  endInput = '';
  error = '';

  ngOnChanges(): void {
    this.error = '';
    if (this.event) {
      this.title = this.event.title;
      this.allDay = this.event.allDay;
      this.notes = this.event.notes ?? '';
      this.startInput = this.format(new Date(this.event.start));
      this.endInput = this.format(new Date(this.event.end));
      return;
    }
    this.title = '';
    this.notes = '';
    this.allDay = false;
    const start = new Date(this.defaultDate);
    start.setHours(9, 0, 0, 0);
    const end = new Date(start);
    end.setHours(10, 0, 0, 0);
    this.startInput = this.format(start);
    this.endInput = this.format(end);
  }

  onAllDayToggle(): void {
    this.startInput = this.format(this.parse(this.startInput));
    this.endInput = this.format(this.parse(this.endInput));
  }

  onSave(e: Event): void {
    e.preventDefault();
    if (!this.title.trim()) {
      this.error = this.t('event.error.title');
      return;
    }
    const toISO = this.allDay ? dateInputToISO : dateTimeInputToISO;
    const start = toISO(this.startInput);
    const end = toISO(this.endInput);
    if (new Date(end).getTime() < new Date(start).getTime()) {
      this.error = this.t('event.error.range');
      return;
    }
    this.save.emit({
      title: this.title.trim(),
      allDay: this.allDay,
      start,
      end,
      notes: this.notes.trim() || undefined,
    });
  }

  t(key: string): string {
    return translate(this.sdk.i18n.locale, key);
  }

  private format(d: Date): string {
    return this.allDay ? toDateInput(d) : toDateTimeInput(d);
  }

  private parse(value: string): Date {
    const d = new Date(value);
    return Number.isNaN(d.getTime()) ? this.defaultDate : d;
  }
}
