import { Injector } from '@angular/core';
import type { FlyspaceSDK } from '@flyspace/sdk';
import { FLYSPACE_SDK } from '@flyspace/sdk';
import { describe, expect, it } from 'vitest';

import { CalendarApi } from '../data/calendar-api';
import type { CalendarEvent } from '../data/event';
import { DashboardWidgetComponent } from './dashboard-widget.component';

type ListFn = () => Promise<CalendarEvent[]>;

function makeWidget(list: ListFn, size: 'small' | 'medium' | 'large' = 'medium'): DashboardWidgetComponent {
  const sdk = { i18n: { locale: 'en' } } as unknown as FlyspaceSDK;
  const api = { list } as unknown as CalendarApi;
  const injector = Injector.create({
    providers: [
      { provide: FLYSPACE_SDK, useValue: sdk },
      { provide: CalendarApi, useValue: api },
      { provide: DashboardWidgetComponent, useClass: DashboardWidgetComponent, deps: [] },
    ],
  });
  const widget = injector.get(DashboardWidgetComponent);
  widget.size = size;
  return widget;
}

function eventAt(id: string, startOffsetMs: number, durationMs = 60 * 60 * 1000): CalendarEvent {
  const start = new Date(Date.now() + startOffsetMs);
  const end = new Date(start.getTime() + durationMs);
  return {
    id,
    title: id,
    allDay: false,
    start: start.toISOString(),
    end: end.toISOString(),
    createdAt: start.toISOString(),
    updatedAt: start.toISOString(),
  };
}

const HOUR = 60 * 60 * 1000;
const DAY = 24 * HOUR;

describe('DashboardWidgetComponent.ngOnInit', () => {
  it('drops already-ended events, sorts by start, and caps at 5 for a medium widget', async () => {
    const events = [
      eventAt('in-3d', 3 * DAY),
      eventAt('past', -2 * DAY), // ended well before now → dropped
      eventAt('in-1d', 1 * DAY),
      eventAt('in-2d', 2 * DAY),
      eventAt('in-5d', 5 * DAY),
      eventAt('in-4d', 4 * DAY),
      eventAt('in-6d', 6 * DAY),
    ];
    const widget = makeWidget(async () => events, 'medium');
    await widget.ngOnInit();

    expect(widget.loading).toBe(false);
    expect(widget.upcoming.map((e) => e.id)).toEqual(['in-1d', 'in-2d', 'in-3d', 'in-4d', 'in-5d']);
  });

  it('caps at 3 for a small widget', async () => {
    const events = [eventAt('a', 1 * DAY), eventAt('b', 2 * DAY), eventAt('c', 3 * DAY), eventAt('d', 4 * DAY)];
    const widget = makeWidget(async () => events, 'small');
    await widget.ngOnInit();

    expect(widget.upcoming.map((e) => e.id)).toEqual(['a', 'b', 'c']);
  });

  it('keeps an event that started but has not yet ended', async () => {
    const ongoing = eventAt('ongoing', -HOUR, 3 * HOUR); // started 1h ago, ends in 2h
    const widget = makeWidget(async () => [ongoing]);
    await widget.ngOnInit();

    expect(widget.upcoming.map((e) => e.id)).toEqual(['ongoing']);
  });

  it('shows nothing (not loading) when the request fails', async () => {
    const widget = makeWidget(async () => {
      throw new Error('proxy down');
    });
    await widget.ngOnInit();

    expect(widget.upcoming).toEqual([]);
    expect(widget.loading).toBe(false);
  });

  it('shows nothing when the backend returns no events', async () => {
    const widget = makeWidget(async () => []);
    await widget.ngOnInit();

    expect(widget.upcoming).toEqual([]);
    expect(widget.loading).toBe(false);
  });
});
