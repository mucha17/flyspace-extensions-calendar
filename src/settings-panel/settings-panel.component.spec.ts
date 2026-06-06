import { Injector } from '@angular/core';
import type { FlyspaceSDK } from '@flyspace/sdk';
import { FLYSPACE_SDK } from '@flyspace/sdk';
import { describe, expect, it } from 'vitest';

import { SettingsPanelComponent } from './settings-panel.component';

interface Harness {
  panel: SettingsPanelComponent;
  sets: { key: string; value: unknown }[];
  toasts: { type: string }[];
}

function makePanel(opts: { stored?: number; getThrows?: boolean; setThrows?: boolean } = {}): Harness {
  const sets: { key: string; value: unknown }[] = [];
  const toasts: { type: string }[] = [];
  const sdk = {
    i18n: { locale: 'en' },
    user: {
      config: {
        get: async (): Promise<number | undefined> => {
          if (opts.getThrows) throw new Error('config unavailable');
          return opts.stored;
        },
        set: async (key: string, value: unknown): Promise<void> => {
          sets.push({ key, value });
          if (opts.setThrows) throw new Error('write failed');
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
      { provide: SettingsPanelComponent, useClass: SettingsPanelComponent, deps: [] },
    ],
  });
  return { panel: injector.get(SettingsPanelComponent), sets, toasts };
}

function changeEvent(value: string): Event {
  return { target: { value } } as unknown as Event;
}

describe('SettingsPanelComponent.ngOnInit', () => {
  it('loads a stored Sunday preference', async () => {
    const { panel } = makePanel({ stored: 0 });
    await panel.ngOnInit();
    expect(panel.weekStart).toBe(0);
  });

  it('defaults to Monday when unset or any non-zero value', async () => {
    const { panel } = makePanel({ stored: undefined });
    await panel.ngOnInit();
    expect(panel.weekStart).toBe(1);
  });

  it('defaults to Monday when the config read fails', async () => {
    const { panel } = makePanel({ getThrows: true });
    await panel.ngOnInit();
    expect(panel.weekStart).toBe(1);
  });
});

describe('SettingsPanelComponent.setWeekStart', () => {
  it('persists Sunday and confirms with a success toast', async () => {
    const { panel, sets, toasts } = makePanel();
    await panel.setWeekStart(changeEvent('0'));

    expect(panel.weekStart).toBe(0);
    expect(sets).toEqual([{ key: 'weekStart', value: 0 }]);
    expect(toasts).toEqual([{ type: 'success' }]);
  });

  it('normalizes any non-zero selection to Monday', async () => {
    const { panel, sets } = makePanel();
    await panel.setWeekStart(changeEvent('5'));

    expect(panel.weekStart).toBe(1);
    expect(sets).toEqual([{ key: 'weekStart', value: 1 }]);
  });

  it('reports a failed save with an error toast', async () => {
    const { panel, toasts } = makePanel({ setThrows: true });
    await panel.setWeekStart(changeEvent('0'));

    expect(toasts).toEqual([{ type: 'error' }]);
  });
});
