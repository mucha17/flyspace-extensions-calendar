import { describe, expect, it } from 'vitest';

import type { CalendarEvent } from './event';
import { addDays, dateInputToISO, monthGrid, occursOn, sameDay, toDateInput } from './date-utils';

function event(start: string, end: string, allDay = false): CalendarEvent {
  return { id: 'e', title: 't', allDay, start, end, createdAt: start, updatedAt: start };
}

describe('monthGrid', () => {
  it('returns 6 full weeks (42 days) starting on the configured weekday', () => {
    const june2026 = new Date(2026, 5, 1); // a Monday
    const mon = monthGrid(june2026, 1);
    expect(mon).toHaveLength(42);
    expect(mon[0].getDay()).toBe(1); // first cell is a Monday
    expect(sameDay(mon[0], june2026)).toBe(true); // June 1 2026 is itself a Monday

    const sun = monthGrid(june2026, 0);
    expect(sun[0].getDay()).toBe(0); // first cell is a Sunday (May 31)
    expect(sameDay(sun[1], june2026)).toBe(true); // June 1 (Monday) is the second cell
  });

  it('pads the leading week with the previous month', () => {
    // July 2026 starts on a Wednesday; a Monday-start grid begins on Mon Jun 29.
    const grid = monthGrid(new Date(2026, 6, 1), 1);
    expect(grid[0].getMonth()).toBe(5); // June
    expect(grid[0].getDate()).toBe(29);
  });
});

describe('occursOn', () => {
  const day = new Date(2026, 5, 2);

  it('matches a timed event on its day', () => {
    expect(occursOn(event('2026-06-02T09:00:00Z', '2026-06-02T10:00:00Z'), day)).toBe(true);
  });

  it('does not match a different day', () => {
    expect(occursOn(event('2026-06-03T09:00:00Z', '2026-06-03T10:00:00Z'), day)).toBe(false);
  });

  it('matches an inner day of a multi-day event', () => {
    // June 1 -> June 3 conference, checked on June 2.
    expect(occursOn(event('2026-06-01T09:00:00Z', '2026-06-03T17:00:00Z'), day)).toBe(true);
  });

  it('matches an all-day event spanning the day', () => {
    const local = (d: number) => dateInputToISO(toDateInput(new Date(2026, 5, d)));
    expect(occursOn(event(local(2), local(2), true), day)).toBe(true);
  });
});

describe('addDays', () => {
  it('crosses a month boundary', () => {
    const d = addDays(new Date(2026, 5, 30), 2);
    expect(d.getMonth()).toBe(6);
    expect(d.getDate()).toBe(2);
  });
});
