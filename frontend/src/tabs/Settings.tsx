import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { SectionCard } from "@/components/SectionCard";
import { SegmentedControl } from "@/components/SegmentedControl";
import { Collapsible } from "@/components/Collapsible";
import { Toast } from "@/components/Toast";
import { useConnectionStore } from "@/stores/connection";
import {
  BulkPingAll,
  LoadSettingsJSON,
  ReloadMihomo,
  SaveSettingsJSON,
  SocketPath,
  StartDaemon,
  StopDaemon,
} from "../wailsjs/go/main/App";

export function SettingsTab() {
  const { conn } = useConnectionStore();
  const [data, setData] = useState<Record<string, unknown>>({});
  const [status, setStatus] = useState("");
  const [socket, setSocket] = useState("");

  useEffect(() => {
    LoadSettingsJSON().then(setData).catch((e) => setStatus(String(e)));
    SocketPath().then(setSocket).catch(() => {});
  }, []);

  async function save(patch: Record<string, unknown>) {
    const next = { ...data, ...patch };
    await SaveSettingsJSON(next);
    setData(next);
    setStatus("Сохранено");
  }

  const mode = (data.serviceMode as string) || "proxy";
  const poolMode =
    (data.aggregationMode as string) === "flow_aggregate" ? "pool" : "single";
  const multipathOn = !!data.multipathEnabled;
  const mpPreset = (data.multipathPreset as string) || "normal";

  return (
    <div className="flex flex-col gap-4 overflow-y-auto p-4">
      {socket ? (
        <p className="text-caption text-muted">
          Сокет демона: <span className="font-mono text-fg">{socket}</span>
        </p>
      ) : null}

      <SectionCard
        title="Режим"
        description="Proxy — системный прокси; VPN — TUN-интерфейс (требует прав администратора)."
      >
        <SegmentedControl
          options={[
            { id: "proxy", label: "Прокси" },
            { id: "vpn", label: "VPN (TUN)" },
          ]}
          value={mode}
          onChange={(m) => save({ serviceMode: m })}
        />
        <label className="block text-sm text-muted">
          Mixed port
          <input
            type="number"
            className="focus-ring mt-1 w-full rounded border border-border bg-surface-2 px-2 py-2"
            value={Number(data.mixedPort) || 2181}
            onChange={(e) => save({ mixedPort: Number(e.target.value) })}
          />
        </label>
      </SectionCard>

      <SectionCard
        title="Пул PROXY_BULK"
        description="Один туннель или load-balance по нескольким ногам."
      >
        <SegmentedControl
          options={[
            { id: "single", label: "Один туннель" },
            { id: "pool", label: "Пул (LB)" },
          ]}
          value={poolMode}
          onChange={(id) =>
            save({
              aggregationMode: id === "pool" ? "flow_aggregate" : "legacy",
              bulkEnabled: id === "pool",
              multipathEnabled: id === "pool" ? true : multipathOn,
            })
          }
        />
        {poolMode === "pool" ? (
          <div className="space-y-2">
            <label className="block text-sm text-muted">
              Стратегия LB
              <select
                className="focus-ring mt-1 w-full rounded border border-border bg-surface-2 px-2 py-2"
                value={(data.bulkLbStrategy as string) || "sticky-sessions"}
                onChange={(e) => save({ bulkLbStrategy: e.target.value })}
              >
                <option value="sticky-sessions">Sticky sessions</option>
                <option value="consistent-hashing">Consistent hash</option>
              </select>
            </label>
            <div className="grid grid-cols-3 gap-2 text-sm">
              <label className="text-muted">
                Min ног
                <input
                  type="number"
                  className="focus-ring mt-1 w-full rounded border border-border bg-surface-2 px-2 py-1"
                  value={Number(data.bulkMinHealthyLegs) || 2}
                  onChange={(e) =>
                    save({ bulkMinHealthyLegs: Number(e.target.value) })
                  }
                />
              </label>
              <label className="text-muted">
                Max ног
                <input
                  type="number"
                  className="focus-ring mt-1 w-full rounded border border-border bg-surface-2 px-2 py-1"
                  value={Number(data.bulkMaxLegs) || 0}
                  onChange={(e) =>
                    save({ bulkMaxLegs: Number(e.target.value) })
                  }
                />
              </label>
              <label className="text-muted">
                Recovery, с
                <input
                  type="number"
                  className="focus-ring mt-1 w-full rounded border border-border bg-surface-2 px-2 py-1"
                  value={Number(data.bulkRecoverySeconds) || 60}
                  onChange={(e) =>
                    save({ bulkRecoverySeconds: Number(e.target.value) })
                  }
                />
              </label>
            </div>
            {conn?.probeText ? (
              <Collapsible title="Таблица ног (из диагностики)">
                <pre className="max-h-32 overflow-y-auto whitespace-pre-wrap text-caption text-muted">
                  {conn.probeText}
                </pre>
              </Collapsible>
            ) : (
              <p className="text-caption text-muted">
                Статус ног — в «Простой → Диагностика» после подключения.
              </p>
            )}
          </div>
        ) : null}
      </SectionCard>

      <SectionCard title="Multipath" description="Goodput-агрегация каналов.">
        <label className="flex min-h-[44px] items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={multipathOn}
            onChange={(e) => save({ multipathEnabled: e.target.checked })}
          />
          Multipath (goodput)
        </label>
        {multipathOn ? (
          <div className="space-y-2">
            <SegmentedControl
              options={[
                { id: "low", label: "low" },
                { id: "normal", label: "normal" },
                { id: "high", label: "high" },
              ]}
              value={mpPreset}
              onChange={(v) => save({ multipathPreset: v })}
            />
            <label className="flex items-center gap-2 text-sm text-muted">
              <input
                type="checkbox"
                checked={!!data.multipathWlEmergencyOnly}
                onChange={(e) =>
                  save({ multipathWlEmergencyOnly: e.target.checked })
                }
              />
              WL emergency only
            </label>
            <label className="flex items-center gap-2 text-sm text-muted">
              <input
                type="checkbox"
                checked={!!data.wlBuiltinConnectEnabled}
                onChange={(e) =>
                  save({ wlBuiltinConnectEnabled: e.target.checked })
                }
              />
              WL builtin connect
            </label>
          </div>
        ) : null}
      </SectionCard>

      <SectionCard title="Демон" description="Управление фоновым процессом muhomor.">
        <div className="flex flex-wrap gap-2">
          <Button
            variant="secondary"
            className="!w-auto"
            onClick={() => ReloadMihomo().then(() => setStatus("Reload mihomo"))}
          >
            Reload mihomo
          </Button>
          <Button
            variant="secondary"
            className="!w-auto"
            onClick={() => BulkPingAll().then((m) => setStatus(m))}
          >
            Ping всех ног
          </Button>
          <Button
            variant="secondary"
            className="!w-auto"
            onClick={() => StopDaemon().then(() => setStatus("Демон остановлен"))}
          >
            Стоп демон
          </Button>
          <Button
            variant="secondary"
            className="!w-auto"
            onClick={() => StartDaemon().then(() => setStatus("Демон запущен"))}
          >
            Старт демон
          </Button>
        </div>
      </SectionCard>

      <SectionCard
        title="Выход UI"
        description="Как завершать работу демона при закрытии интерфейса."
      >
        <label className="flex min-h-[44px] items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={data.stopDaemonOnExit === undefined ? true : !!data.stopDaemonOnExit}
            onChange={(e) => save({ stopDaemonOnExit: e.target.checked })}
          />
          Останавливать демон при выходе из UI
        </label>
      </SectionCard>

      <Collapsible title="Дополнительно">
        <label className="flex min-h-[44px] items-center gap-2 text-sm text-muted">
          <input
            type="checkbox"
            checked={!!data.uiKeepErrorsOnScreen}
            onChange={(e) => save({ uiKeepErrorsOnScreen: e.target.checked })}
          />
          Оставлять ошибки на экране
        </label>
      </Collapsible>

      <Toast text={status} onDismiss={() => setStatus("")} />
    </div>
  );
}
