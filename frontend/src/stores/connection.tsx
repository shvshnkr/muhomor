import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { EventsOn } from "../wailsjs/runtime/runtime";
import {
  ConnectionDTO,
  GetConnectionSnapshot,
  GetSettingsSnapshot,
  SettingsDTO,
} from "../wailsjs/go/main/App";

export type ToastMessage = { level: string; text: string };

type ConnectionContextValue = {
  conn: ConnectionDTO | null;
  settings: SettingsDTO | null;
  toast: ToastMessage | null;
  clearToast: () => void;
};

const ConnectionContext = createContext<ConnectionContextValue | null>(null);

function stripConnectionErrors(dto: ConnectionDTO): ConnectionDTO {
  return {
    ...dto,
    errorText: "",
    activityText: "",
    pingError: false,
    busy: false,
    connecting: false,
    statusTitle: dto.connected ? "Подключено" : "Отключено",
  };
}

/** Single SSE subscription for the whole app (survives simple ↔ extended switches). */
export function ConnectionProvider({ children }: { children: ReactNode }) {
  const [conn, setConn] = useState<ConnectionDTO | null>(null);
  const [settings, setSettings] = useState<SettingsDTO | null>(null);
  const [toast, setToast] = useState<ToastMessage | null>(null);
  const quittingRef = useRef(false);

  useEffect(() => {
    let cancelled = false;
    Promise.all([GetConnectionSnapshot(), GetSettingsSnapshot()])
      .then(([c, s]) => {
        if (!cancelled && !quittingRef.current) {
          setConn(c);
          setSettings(s);
        }
      })
      .catch((e) => console.error("connection snapshot", e));

    const offShutdown = EventsOn("shutdown", () => {
      quittingRef.current = true;
      setToast(null);
      setConn((prev) => (prev ? stripConnectionErrors(prev) : prev));
    });

    const offConn = EventsOn("connection", (data) => {
      if (quittingRef.current) return;
      setConn(data as ConnectionDTO);
    });
    const offSet = EventsOn("settings", (data) => {
      if (quittingRef.current) return;
      setSettings(data as SettingsDTO);
    });
    const offToast = EventsOn("toast", (data) => {
      if (quittingRef.current) return;
      const t = data as ToastMessage;
      if (t?.text && t.level !== "error") setToast(t);
    });
    return () => {
      cancelled = true;
      offShutdown();
      offConn();
      offSet();
      offToast();
    };
  }, []);

  return (
    <ConnectionContext.Provider
      value={{ conn, settings, toast, clearToast: () => setToast(null) }}
    >
      {children}
    </ConnectionContext.Provider>
  );
}

export function useConnectionStore(): ConnectionContextValue {
  const ctx = useContext(ConnectionContext);
  if (!ctx) {
    throw new Error("useConnectionStore requires ConnectionProvider");
  }
  return ctx;
}
