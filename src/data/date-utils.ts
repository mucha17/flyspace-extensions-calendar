import type { CalendarEvent } from './event';

/** Midnight at the start of the given day, in local time. */
export function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

export function addDays(d: Date, n: number): Date {
  const r = new Date(d);
  r.setDate(r.getDate() + n);
  return r;
}

export function addMonths(d: Date, n: number): Date {
  return new Date(d.getFullYear(), d.getMonth() + n, 1);
}

export function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

/**
 * The 6×7 grid of days for the month containing `month`, padded with the surrounding days so each
 * row is a full week. `weekStart` is 0 (Sunday) or 1 (Monday).
 */
export function monthGrid(month: Date, weekStart: 0 | 1): Date[] {
  const first = new Date(month.getFullYear(), month.getMonth(), 1);
  const offset = (first.getDay() - weekStart + 7) % 7;
  const gridStart = addDays(first, -offset);
  return Array.from({ length: 42 }, (_, i) => addDays(gridStart, i));
}

/** Whether an event covers any part of the given day (handles all-day and multi-day events). */
export function occursOn(event: CalendarEvent, day: Date): boolean {
  const dayStart = startOfDay(day).getTime();
  const dayEnd = addDays(startOfDay(day), 1).getTime();
  const start = new Date(event.start).getTime();
  const end = new Date(event.end).getTime();
  return start < dayEnd && end >= dayStart;
}

/** Format a Date as the value of an <input type="datetime-local"> (local, minute precision). */
export function toDateTimeInput(d: Date): string {
  return `${dateInput(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** Format a Date as the value of an <input type="date">. */
export function toDateInput(d: Date): string {
  return dateInput(d);
}

/** Parse an <input type="datetime-local"> value (local time) to an ISO string. */
export function dateTimeInputToISO(value: string): string {
  return new Date(value).toISOString();
}

/** Parse an <input type="date"> value to an ISO string at local midnight. */
export function dateInputToISO(value: string): string {
  const [y, m, d] = value.split('-').map(Number);
  return new Date(y, m - 1, d).toISOString();
}

function dateInput(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`;
}
