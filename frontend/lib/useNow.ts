import { useSyncExternalStore } from "react";

// One shared clock for every time on the page: a single interval, not one per row.
// On the server and during hydration there is no "now" (null), so the HTML
// stays the same however long it sits in a cache.
const TICK_MS = 30_000;

let now: number | null = null;
let timer: ReturnType<typeof setInterval> | undefined;
const listeners = new Set<() => void>();

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  if (!timer) {
    timer = setInterval(() => {
      now = Date.now();
      listeners.forEach((l) => l());
    }, TICK_MS);
  }
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) {
      clearInterval(timer);
      timer = undefined;
      now = null; // the next subscriber starts from the current time
    }
  };
}

function getSnapshot(): number {
  return (now ??= Date.now());
}

function getServerSnapshot(): null {
  return null;
}

/** The current time in ms in the browser, updated every 30 s; null on the server. */
export function useNow(): number | null {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
