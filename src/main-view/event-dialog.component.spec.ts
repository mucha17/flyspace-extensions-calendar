import { Injector } from '@angular/core';
import type { FlyspaceSDK } from '@flyspace/sdk';
import { FLYSPACE_SDK } from '@flyspace/sdk';
import { describe, expect, it } from 'vitest';

import type { CalendarEvent, EventInput } from '../data/event';
import { dateInputToISO, dateTimeInputToISO, toDateTimeInput } from '../data/date-utils';
import { EventDialogComponent } from './event-dialog.component';

function makeDialog(locale = 'en'): EventDialogComponent {
  const sdk = { i18n: { locale } } as unknown as FlyspaceSDK;
  const injector = Injector.create({
    providers: [
      { provide: FLYSPACE_SDK, useValue: sdk },
      { provide: EventDialogComponent, useClass: EventDialogComponent, deps: [] },
    ],
  });
  return injector.get(EventDialogComponent);
}

/** A submit event whose default the dialog suppresses. */
function submitEvent(): Event {
  return { preventDefault: () => undefined } as unknown as Event;
}

function captureSaves(dialog: EventDialogComponent): EventInput[] {
  const saved: EventInput[] = [];
  dialog.save.subscribe((v) => saved.push(v));
  return saved;
}

describe('EventDialogComponent.ngOnChanges', () => {
  it('seeds a 9–10 default window for a new (null) event', () => {
    const dialog = makeDialog();
    dialog.event = null;
    dialog.defaultDate = new Date(2026, 5, 2);
    dialog.ngOnChanges();

    expect(dialog.title).toBe('');
    expect(dialog.allDay).toBe(false);
    expect(dialog.notes).toBe('');
    expect(dialog.startInput).toBe('2026-06-02T09:00');
    expect(dialog.endInput).toBe('2026-06-02T10:00');
    expect(dialog.error).toBe('');
  });

  it('populates the form from an existing event', () => {
    const dialog = makeDialog();
    const event: CalendarEvent = {
      id: 'e1',
      title: 'Standup',
      allDay: false,
      start: new Date(2026, 5, 2, 9, 0).toISOString(),
      end: new Date(2026, 5, 2, 9, 30).toISOString(),
      notes: 'bring coffee',
      createdAt: '2026-06-01T00:00:00Z',
      updatedAt: '2026-06-01T00:00:00Z',
    };
    dialog.event = event;
    dialog.ngOnChanges();

    expect(dialog.title).toBe('Standup');
    expect(dialog.allDay).toBe(false);
    expect(dialog.notes).toBe('bring coffee');
    expect(dialog.startInput).toBe(toDateTimeInput(new Date(event.start)));
    expect(dialog.endInput).toBe(toDateTimeInput(new Date(event.end)));
  });

  it('treats a missing notes field as an empty string', () => {
    const dialog = makeDialog();
    dialog.event = {
      id: 'e1',
      title: 'x',
      allDay: true,
      start: new Date(2026, 5, 2).toISOString(),
      end: new Date(2026, 5, 2).toISOString(),
      createdAt: '2026-06-01T00:00:00Z',
      updatedAt: '2026-06-01T00:00:00Z',
    };
    dialog.ngOnChanges();
    expect(dialog.notes).toBe('');
  });
});

describe('EventDialogComponent.onSave validation', () => {
  it('rejects an empty title without emitting', () => {
    const dialog = makeDialog();
    const saved = captureSaves(dialog);
    dialog.title = '   ';
    dialog.startInput = '2026-06-02T09:00';
    dialog.endInput = '2026-06-02T10:00';
    dialog.onSave(submitEvent());

    expect(dialog.error).toBe('A title is required.');
    expect(saved).toHaveLength(0);
  });

  it('rejects an end before the start without emitting', () => {
    const dialog = makeDialog();
    const saved = captureSaves(dialog);
    dialog.title = 'Standup';
    dialog.startInput = '2026-06-02T10:00';
    dialog.endInput = '2026-06-02T09:00';
    dialog.onSave(submitEvent());

    expect(dialog.error).toBe('End must be on or after start.');
    expect(saved).toHaveLength(0);
  });

  it('allows an end equal to the start', () => {
    const dialog = makeDialog();
    const saved = captureSaves(dialog);
    dialog.title = 'Marker';
    dialog.startInput = '2026-06-02T09:00';
    dialog.endInput = '2026-06-02T09:00';
    dialog.onSave(submitEvent());

    expect(saved).toHaveLength(1);
  });
});

describe('EventDialogComponent.onSave emission', () => {
  it('emits a trimmed, ISO-normalized timed event', () => {
    const dialog = makeDialog();
    const saved = captureSaves(dialog);
    dialog.title = '  Standup  ';
    dialog.allDay = false;
    dialog.notes = '  bring coffee  ';
    dialog.startInput = '2026-06-02T09:00';
    dialog.endInput = '2026-06-02T10:00';
    dialog.onSave(submitEvent());

    expect(saved[0]).toEqual({
      title: 'Standup',
      allDay: false,
      start: dateTimeInputToISO('2026-06-02T09:00'),
      end: dateTimeInputToISO('2026-06-02T10:00'),
      notes: 'bring coffee',
    });
  });

  it('drops blank notes to undefined', () => {
    const dialog = makeDialog();
    const saved = captureSaves(dialog);
    dialog.title = 'Standup';
    dialog.notes = '   ';
    dialog.startInput = '2026-06-02T09:00';
    dialog.endInput = '2026-06-02T10:00';
    dialog.onSave(submitEvent());

    expect(saved[0].notes).toBeUndefined();
  });

  it('uses date-only ISO parsing for an all-day event', () => {
    const dialog = makeDialog();
    const saved = captureSaves(dialog);
    dialog.title = 'Conference';
    dialog.allDay = true;
    dialog.startInput = '2026-06-02';
    dialog.endInput = '2026-06-04';
    dialog.onSave(submitEvent());

    expect(saved[0]).toEqual({
      title: 'Conference',
      allDay: true,
      start: dateInputToISO('2026-06-02'),
      end: dateInputToISO('2026-06-04'),
      notes: undefined,
    });
  });
});

describe('EventDialogComponent.onAllDayToggle', () => {
  it('reformats the inputs from datetime to date when switched to all-day', () => {
    const dialog = makeDialog();
    dialog.allDay = false;
    dialog.startInput = '2026-06-02T09:00';
    dialog.endInput = '2026-06-02T10:00';

    dialog.allDay = true;
    dialog.onAllDayToggle();

    expect(dialog.startInput).toBe('2026-06-02');
    expect(dialog.endInput).toBe('2026-06-02');
  });
});
