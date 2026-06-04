import { Injectable, inject } from '@angular/core';
import { FLYSPACE_SDK } from '@flyspace/sdk';

import type { CalendarEvent, EventInput } from './event';

/**
 * Talks to the calendar's own backend through the platform's mediated proxy. Every call goes
 * `sdk.backend.request` → core → the calendar backend; the browser never holds a token and never
 * reaches the backend directly. Provided on the mounted route, where the scoped FLYSPACE_SDK lives.
 */
@Injectable()
export class CalendarApi {
  private readonly sdk = inject(FLYSPACE_SDK);

  list(fromISO: string, toISO: string): Promise<CalendarEvent[]> {
    return this.sdk.backend.request<CalendarEvent[]>('/events', { query: { from: fromISO, to: toISO } });
  }

  create(input: EventInput): Promise<CalendarEvent> {
    return this.sdk.backend.request<CalendarEvent>('/events', { method: 'POST', body: input });
  }

  replace(id: string, input: EventInput): Promise<CalendarEvent> {
    return this.sdk.backend.request<CalendarEvent>(`/events/${encodeURIComponent(id)}`, { method: 'PUT', body: input });
  }

  remove(id: string): Promise<void> {
    return this.sdk.backend.request<void>(`/events/${encodeURIComponent(id)}`, { method: 'DELETE' });
  }
}
