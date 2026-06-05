import { Injector } from '@angular/core';
import type { BackendRequestInit, FlyspaceSDK } from '@flyspace/sdk';
import { FLYSPACE_SDK } from '@flyspace/sdk';
import { beforeEach, describe, expect, it } from 'vitest';

import { CalendarApi } from './calendar-api';
import type { CalendarEvent, EventInput } from './event';

interface RecordedCall {
  path: string;
  init?: BackendRequestInit;
}

/**
 * A fake SDK that records every `backend.request` and returns a queued response. The calendar's
 * only door to the platform is this method, so stubbing it is the seam the client is tested at —
 * we assert the path/method/body/query the client builds and that it returns what the proxy gives
 * back, never touching the network.
 */
function fakeSdk(response: unknown = undefined): { sdk: FlyspaceSDK; calls: RecordedCall[] } {
  const calls: RecordedCall[] = [];
  const sdk = {
    backend: {
      request: async <T>(path: string, init?: BackendRequestInit): Promise<T> => {
        calls.push({ path, init });
        return response as T;
      },
    },
  } as unknown as FlyspaceSDK;
  return { sdk, calls };
}

function makeApi(sdk: FlyspaceSDK): CalendarApi {
  const injector = Injector.create({
    providers: [
      { provide: FLYSPACE_SDK, useValue: sdk },
      { provide: CalendarApi, useClass: CalendarApi, deps: [] },
    ],
  });
  return injector.get(CalendarApi);
}

const sampleEvent: CalendarEvent = {
  id: 'evt-1',
  title: 'Standup',
  allDay: false,
  start: '2026-06-02T09:00:00Z',
  end: '2026-06-02T09:15:00Z',
  createdAt: '2026-06-01T00:00:00Z',
  updatedAt: '2026-06-01T00:00:00Z',
};

const sampleInput: EventInput = {
  title: 'Standup',
  allDay: false,
  start: '2026-06-02T09:00:00Z',
  end: '2026-06-02T09:15:00Z',
};

describe('CalendarApi', () => {
  let sdk: FlyspaceSDK;
  let calls: RecordedCall[];

  describe('list', () => {
    beforeEach(() => {
      ({ sdk, calls } = fakeSdk([sampleEvent]));
    });

    it('GETs /events with the range as query params and returns the parsed events', async () => {
      const api = makeApi(sdk);
      const result = await api.list('2026-06-01T00:00:00Z', '2026-06-30T23:59:59Z');

      expect(result).toEqual([sampleEvent]);
      expect(calls).toHaveLength(1);
      expect(calls[0].path).toBe('/events');
      expect(calls[0].init?.method ?? 'GET').toBe('GET');
      expect(calls[0].init?.query).toEqual({
        from: '2026-06-01T00:00:00Z',
        to: '2026-06-30T23:59:59Z',
      });
    });
  });

  describe('create', () => {
    beforeEach(() => {
      ({ sdk, calls } = fakeSdk(sampleEvent));
    });

    it('POSTs /events with the input as the body and returns the created event', async () => {
      const api = makeApi(sdk);
      const result = await api.create(sampleInput);

      expect(result).toEqual(sampleEvent);
      expect(calls[0].path).toBe('/events');
      expect(calls[0].init?.method).toBe('POST');
      expect(calls[0].init?.body).toEqual(sampleInput);
    });
  });

  describe('replace', () => {
    beforeEach(() => {
      ({ sdk, calls } = fakeSdk(sampleEvent));
    });

    it('PUTs /events/{id} with the input as the body', async () => {
      const api = makeApi(sdk);
      await api.replace('evt-1', sampleInput);

      expect(calls[0].path).toBe('/events/evt-1');
      expect(calls[0].init?.method).toBe('PUT');
      expect(calls[0].init?.body).toEqual(sampleInput);
    });

    it('encodes an id with reserved characters into the path', async () => {
      const api = makeApi(sdk);
      await api.replace('a/b?c#d', sampleInput);

      expect(calls[0].path).toBe('/events/a%2Fb%3Fc%23d');
    });
  });

  describe('remove', () => {
    beforeEach(() => {
      ({ sdk, calls } = fakeSdk(undefined));
    });

    it('DELETEs /events/{id}', async () => {
      const api = makeApi(sdk);
      await api.remove('evt-1');

      expect(calls[0].path).toBe('/events/evt-1');
      expect(calls[0].init?.method).toBe('DELETE');
    });

    it('encodes the id into the path', async () => {
      const api = makeApi(sdk);
      await api.remove('a/b');

      expect(calls[0].path).toBe('/events/a%2Fb');
    });
  });

  it('propagates a rejected backend request to the caller', async () => {
    const sdk = {
      backend: {
        request: async () => {
          throw new Error('proxy 502');
        },
      },
    } as unknown as FlyspaceSDK;
    const api = makeApi(sdk);

    await expect(api.list('a', 'b')).rejects.toThrow('proxy 502');
  });
});
