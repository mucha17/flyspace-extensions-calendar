import { Injector } from '@angular/core';
import type { FlyspaceSDK } from '@flyspace/sdk';
import { FLYSPACE_SDK } from '@flyspace/sdk';
import { beforeEach, describe, expect, it } from 'vitest';

import { CalendarApi } from '../data/calendar-api';
import type { CalendarEvent, EventInput } from '../data/event';
import { addDays, sameDay } from '../data/date-utils';
import { MainViewComponent } from './main-view.component';

interface ApiRecorder {
  list: [string, string][];
  create: EventInput[];
  replace: [string, EventInput][];
  remove: string[];
}

interface Harness {
  view: MainViewComponent;
  api: ApiRecorder;
  toasts: { type: string }[];
}

function makeView(opts: {
  events?: CalendarEvent[];
  listThrows?: boolean;
  writeThrows?: boolean;
  config?: number | undefined;
  configThrows?: boolean;
} = {}): Harness {
  const api: ApiRecorder = { list: [], create: [], replace: [], remove: [] };
  const toasts: { type: string }[] = [];

  const apiImpl = {
    list: async (from: string, to: string): Promise<CalendarEvent[]> => {
      api.list.push([from, to]);
      if (opts.listThrows) throw new Error('list failed');
      return opts.events ?? [];
    },
    create: async (input: EventInput): Promise<CalendarEvent> => {
      api.create.push(input);
      if (opts.writeThrows) throw new Error('create failed');
      return {} as CalendarEvent;
    },
    replace: async (id: string, input: EventInput): Promise<CalendarEvent> => {
      api.replace.push([id, input]);
      if (opts.writeThrows) throw new Error('replace failed');
      return {} as CalendarEvent;
    },
    remove: async (id: string): Promise<void> => {
      api.remove.push(id);
      if (opts.writeThrows) throw new Error('remove failed');
    },
  } as unknown as CalendarApi;

  const sdk = {
    i18n: { locale: 'en' },
    user: {
      config: {
        get: async (): Promise<number | undefined> => {
          if (opts.configThrows) throw new Error('config unavailable');
          return opts.config;
        },
      },
    },
    ui: {
      toast: (_msg: unknown, type: 'info' | 'success' | 'error'): void => {
        toasts.push({ type });
      },
    },
  } as unknown as FlyspaceSDK;

  const injector = Injector.create({
    providers: [
      { provide: FLYSPACE_SDK, useValue: sdk },
      { provide: CalendarApi, useValue: apiImpl },
      { provide: MainViewComponent, useClass: MainViewComponent, deps: [] },
    ],
  });
  return { view: injector.get(MainViewComponent), api, toasts };
}

function event(id: string, start: string, end: string, allDay = false): CalendarEvent {
  return { id, title: id, allDay, start, end, createdAt: start, updatedAt: start };
}

describe('MainViewComponent.ngOnInit', () => {
  it('defaults the week to Monday and builds a 6-week grid', async () => {
    const { view } = makeView({ config: undefined });
    await view.ngOnInit();

    expect(view.weekStart).toBe(1);
    expect(view.grid).toHaveLength(42);
    expect(view.weekdays).toHaveLength(7);
    expect(view.monthLabel).not.toBe('');
  });

  it('honors a stored Sunday week start', async () => {
    const { view } = makeView({ config: 0 });
    await view.ngOnInit();
    expect(view.weekStart).toBe(0);
  });

  it('falls back to Monday when config reads fail', async () => {
    const { view } = makeView({ configThrows: true });
    await view.ngOnInit();
    expect(view.weekStart).toBe(1);
  });

  it('requests the visible range and clears the error flag on success', async () => {
    const { view, api } = makeView({ events: [] });
    await view.ngOnInit();

    expect(api.list).toHaveLength(1);
    expect(view.loadError).toBe(false);
  });

  it('flags a load error and empties events when the request fails', async () => {
    const { view } = makeView({ listThrows: true });
    await view.ngOnInit();

    expect(view.loadError).toBe(true);
    expect(view.events).toEqual([]);
  });
});

describe('MainViewComponent day helpers', () => {
  let harness: Harness;
  beforeEach(() => {
    harness = makeView();
  });

  it('eventsOn returns only events covering the day (incl. multi-day spans)', () => {
    harness.view.events = [
      event('on-2nd', '2026-06-02T09:00:00Z', '2026-06-02T10:00:00Z'),
      event('on-5th', '2026-06-05T09:00:00Z', '2026-06-05T10:00:00Z'),
      event('span', '2026-06-01T09:00:00Z', '2026-06-03T17:00:00Z'),
    ];
    const ids = harness.view.eventsOn(new Date(2026, 5, 2)).map((e) => e.id);
    expect(ids).toEqual(['on-2nd', 'span']);
  });

  it('inMonth reflects the currently shown month', () => {
    harness.view.month = new Date(2026, 5, 1);
    expect(harness.view.inMonth(new Date(2026, 5, 15))).toBe(true);
    expect(harness.view.inMonth(new Date(2026, 6, 1))).toBe(false);
  });

  it('isToday matches the current day only', () => {
    expect(harness.view.isToday(new Date())).toBe(true);
    expect(harness.view.isToday(addDays(new Date(), 1))).toBe(false);
  });
});

describe('MainViewComponent dialog state', () => {
  it('openNew opens a blank dialog anchored to the clicked day', () => {
    const { view } = makeView();
    const day = new Date(2026, 5, 10);
    view.openNew(day);

    expect(view.dialogOpen).toBe(true);
    expect(view.editing).toBeNull();
    expect(view.selectedDate).toBe(day);
  });

  it('openEdit opens the dialog for the event and stops propagation', () => {
    const { view } = makeView();
    const target = event('e1', '2026-06-02T09:00:00Z', '2026-06-02T10:00:00Z');
    let stopped = false;
    view.openEdit(target, { stopPropagation: () => (stopped = true) } as unknown as Event);

    expect(stopped).toBe(true);
    expect(view.dialogOpen).toBe(true);
    expect(view.editing).toBe(target);
  });

  it('closeDialog clears the editing state', () => {
    const { view } = makeView();
    view.editing = event('e1', '2026-06-02T09:00:00Z', '2026-06-02T10:00:00Z');
    view.dialogOpen = true;
    view.closeDialog();

    expect(view.dialogOpen).toBe(false);
    expect(view.editing).toBeNull();
  });
});

describe('MainViewComponent.onSave / onDelete', () => {
  const input: EventInput = {
    title: 'New',
    allDay: false,
    start: '2026-06-02T09:00:00Z',
    end: '2026-06-02T10:00:00Z',
  };

  it('creates when there is no editing event, then closes and refreshes', async () => {
    const { view, api } = makeView();
    view.editing = null;
    await view.onSave(input);

    expect(api.create).toEqual([input]);
    expect(api.replace).toHaveLength(0);
    expect(view.dialogOpen).toBe(false);
    expect(api.list).toHaveLength(1); // refresh after save
  });

  it('replaces when editing an existing event', async () => {
    const { view, api } = makeView();
    view.editing = event('e1', '2026-06-02T09:00:00Z', '2026-06-02T10:00:00Z');
    await view.onSave(input);

    expect(api.replace).toEqual([['e1', input]]);
    expect(api.create).toHaveLength(0);
  });

  it('toasts an error and keeps the dialog open when a save fails', async () => {
    const { view, toasts } = makeView({ writeThrows: true });
    view.editing = null;
    view.dialogOpen = true;
    await view.onSave(input);

    expect(toasts).toEqual([{ type: 'error' }]);
    expect(view.dialogOpen).toBe(true);
  });

  it('deletes, then closes and refreshes', async () => {
    const { view, api } = makeView();
    await view.onDelete('e1');

    expect(api.remove).toEqual(['e1']);
    expect(view.dialogOpen).toBe(false);
    expect(api.list).toHaveLength(1);
  });

  it('toasts an error when a delete fails', async () => {
    const { view, toasts } = makeView({ writeThrows: true });
    await view.onDelete('e1');

    expect(toasts).toEqual([{ type: 'error' }]);
  });
});

describe('MainViewComponent navigation', () => {
  it('shift moves the month and reloads', async () => {
    const { view, api } = makeView();
    view.month = new Date(2026, 5, 1);
    await view.shift(1);

    expect(view.month.getMonth()).toBe(6); // July
    expect(api.list).toHaveLength(1);
  });

  it('goToday returns to the current month', async () => {
    const { view } = makeView();
    view.month = new Date(2020, 0, 1);
    await view.goToday();

    const now = new Date();
    expect(sameDay(view.month, new Date(now.getFullYear(), now.getMonth(), 1))).toBe(true);
  });
});
