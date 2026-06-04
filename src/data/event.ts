/** A calendar event as returned by the backend (times are RFC 3339 strings). */
export interface CalendarEvent {
  id: string;
  title: string;
  allDay: boolean;
  start: string;
  end: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

/** The mutable fields sent when creating or replacing an event. */
export interface EventInput {
  title: string;
  allDay: boolean;
  start: string;
  end: string;
  notes?: string;
}
