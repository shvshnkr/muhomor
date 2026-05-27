// Stub for `wails dev` / IDE; replaced at runtime by Wails.
export function EventsOn(
  eventName: string,
  callback: (...data: unknown[]) => void
): () => void {
  const w = window as Window & {
    runtime?: { EventsOn: typeof EventsOn };
  };
  if (w.runtime?.EventsOn) {
    return w.runtime.EventsOn(eventName, callback);
  }
  return () => {};
}

export function EventsOff(eventName: string, ..._args: unknown[]): void {
  const w = window as Window & { runtime?: { EventsOff: typeof EventsOff } };
  w.runtime?.EventsOff?.(eventName);
}

type WailsAppBindings = Record<string, (...a: unknown[]) => Promise<unknown>>;

function wailsAppBindings(): WailsAppBindings | undefined {
  const go = (
    window as Window & {
      go?: Record<string, { App?: WailsAppBindings }>;
    }
  ).go;
  if (!go) return undefined;
  // Bound struct lives in package wailsapp (internal/ui/wailsapp).
  return go.wailsapp?.App ?? go.main?.App;
}

export function Call(method: string, ...args: unknown[]): Promise<unknown> {
  const fn = wailsAppBindings()?.[method];
  if (fn) {
    return fn(...args);
  }
  return Promise.reject(new Error(`Wails binding ${method} not available`));
}

export function WindowSetSize(width: number, height: number): void {
  const w = window as Window & {
    runtime?: { WindowSetSize: (w: number, h: number) => void };
  };
  w.runtime?.WindowSetSize?.(width, height);
}

export function WindowCenter(): void {
  const w = window as Window & {
    runtime?: { WindowCenter: () => void };
  };
  w.runtime?.WindowCenter?.();
}

export function WindowMinimise(): void {
  const w = window as Window & {
    runtime?: { WindowMinimise: () => void };
  };
  w.runtime?.WindowMinimise?.();
}

export function WindowHide(): void {
  const w = window as Window & {
    runtime?: { WindowHide: () => void };
  };
  w.runtime?.WindowHide?.();
}

export function WindowToggleMaximise(): void {
  const w = window as Window & {
    runtime?: { WindowToggleMaximise: () => void };
  };
  w.runtime?.WindowToggleMaximise?.();
}

export function Quit(): void {
  const w = window as Window & {
    runtime?: { Quit: () => void };
  };
  w.runtime?.Quit?.();
}
