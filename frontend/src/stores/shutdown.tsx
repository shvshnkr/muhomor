import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";
import type { ShutdownState } from "@/components/ShutdownOverlay";

type ShutdownContextValue = {
  shuttingDown: boolean;
  shutdown: ShutdownState | null;
};

const ShutdownContext = createContext<ShutdownContextValue>({
  shuttingDown: false,
  shutdown: null,
});

function parseShutdownPayload(data: unknown): ShutdownState | null {
  const raw = (Array.isArray(data) ? data[0] : data) as Record<
    string,
    unknown
  > | null;
  if (!raw || typeof raw !== "object") return null;
  const active = raw.active === true || raw.Active === true;
  if (!active) return null;
  const phase = String(raw.phase ?? raw.Phase ?? "");
  const message = String(raw.message ?? raw.Message ?? "Завершение…");
  return { active: true, phase, message };
}

export function ShutdownProvider({ children }: { children: ReactNode }) {
  const [shutdown, setShutdown] = useState<ShutdownState | null>(null);

  useEffect(() => {
    const off = EventsOn("shutdown", (data) => {
      const parsed = parseShutdownPayload(data);
      if (parsed) setShutdown(parsed);
    });
    return off;
  }, []);

  return (
    <ShutdownContext.Provider
      value={{ shuttingDown: !!shutdown?.active, shutdown }}
    >
      {children}
    </ShutdownContext.Provider>
  );
}

export function useShutdown(): ShutdownContextValue {
  return useContext(ShutdownContext);
}
