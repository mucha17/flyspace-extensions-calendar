import { describe, expect, it } from 'vitest';

import type { CalendarEvent } from './event';
import {
  addDays,
  addMonths,
  dateInputToISO,
  dateTimeInputToISO,
  monthGrid,
  occursOn,
  sameDay,
  startOfDay,
  toDateInput,
  toDateTimeInput,
} from './date-utils';

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

  it('does not mutate its argument', () => {
    const original = new Date(2026, 5, 30);
    addDays(original, 5);
    expect(original.getDate()).toBe(30);
  });
});

describe('startOfDay', () => {
  it('zeroes the time component, keeping the local date', () => {
    const d = startOfDay(new Date(2026, 5, 2, 14, 37, 9, 500));
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(5);
    expect(d.getDate()).toBe(2);
    expect(d.getHours()).toBe(0);
    expect(d.getMinutes()).toBe(0);
    expect(d.getSeconds()).toBe(0);
    expect(d.getMilliseconds()).toBe(0);
  });
});

describe('addMonths', () => {
  it('returns the first of the target month', () => {
    const d = addMonths(new Date(2026, 5, 17), 2);
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(7); // August
    expect(d.getDate()).toBe(1);
  });

  it('rolls into the next year and back', () => {
    expect(addMonths(new Date(2026, 11, 10), 1).getFullYear()).toBe(2027);
    expect(addMonths(new Date(2026, 11, 10), 1).getMonth()).toBe(0);
    expect(addMonths(new Date(2026, 0, 10), -1).getFullYear()).toBe(2025);
    expect(addMonths(new Date(2026, 0, 10), -1).getMonth()).toBe(11);
  });
});

describe('sameDay', () => {
  it('ignores the time of day', () => {
    expect(sameDay(new Date(2026, 5, 2, 1, 0), new Date(2026, 5, 2, 23, 0))).toBe(true);
  });

  it('distinguishes the same date in different months/years', () => {
    expect(sameDay(new Date(2026, 5, 2), new Date(2026, 6, 2))).toBe(false);
    expect(sameDay(new Date(2026, 5, 2), new Date(2025, 5, 2))).toBe(false);
  });
});

describe('date input round-trips (local time)', () => {
  it('toDateInput → dateInputToISO returns local midnight of the same calendar day', () => {
    const d = new Date(2026, 5, 2, 14, 30);
    const iso = dateInputToISO(toDateInput(d));
    const back = new Date(iso);
    expect(sameDay(back, d)).toBe(true);
    expect(back.getHours()).toBe(0);
    expect(back.getMinutes()).toBe(0);
  });

  it('toDateTimeInput → dateTimeInputToISO preserves the instant to the minute', () => {
    const d = new Date(2026, 5, 2, 14, 37);
    const back = new Date(dateTimeInputToISO(toDateTimeInput(d)));
    expect(back.getFullYear()).toBe(2026);
    expect(back.getMonth()).toBe(5);
    expect(back.getDate()).toBe(2);
    expect(back.getHours()).toBe(14);
    expect(back.getMinutes()).toBe(37);
  });

  it('zero-pads month, day, hour, and minute', () => {
    expect(toDateInput(new Date(2026, 0, 3))).toBe('2026-01-03');
    expect(toDateTimeInput(new Date(2026, 0, 3, 4, 5))).toBe('2026-01-03T04:05');
  });
});
