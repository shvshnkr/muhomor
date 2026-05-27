import { useEffect, useRef, useState } from "react";

/** Elapsed seconds while connected; resets on disconnect. */
export function useSessionTimer(connected: boolean): number {
  const [elapsed, setElapsed] = useState(0);
  const startedAt = useRef<number | null>(null);

  useEffect(() => {
    if (!connected) {
      startedAt.current = null;
      setElapsed(0);
      return;
    }
    if (startedAt.current === null) {
      startedAt.current = Date.now();
    }
    const tick = () => {
      if (startedAt.current !== null) {
        setElapsed(Math.floor((Date.now() - startedAt.current) / 1000));
      }
    };
    tick();
    const id = window.setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [connected]);

  return elapsed;
}
