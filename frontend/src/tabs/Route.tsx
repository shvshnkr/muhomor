import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { SectionCard } from "@/components/SectionCard";
import { SegmentedControl } from "@/components/SegmentedControl";
import { Toast } from "@/components/Toast";
import { GetRouteQuick, SetRouteQuick } from "../wailsjs/go/main/App";

const PRESETS = [
  {
    id: "0",
    label: "0 · Ручной",
    hint: "Правила только из вашего конфига, без автопресетов.",
  },
  {
    id: "1",
    label: "1 · RU напрямую",
    hint: "Трафик .ru и локальные сети без прокси.",
  },
  {
    id: "2",
    label: "2 · RU/AI через прокси",
    hint: "Заблокированные .ru и AI-сервисы через туннель.",
  },
  {
    id: "3",
    label: "3 · WG over WL",
    hint: "Private WG через прокси-туннель (WL tunnel).",
  },
] as const;

export function RouteTab() {
  const [idx, setIdx] = useState("0");
  const [status, setStatus] = useState("");

  useEffect(() => {
    GetRouteQuick()
      .then((n) => setIdx(String(n)))
      .catch(console.error);
  }, []);

  return (
    <div className="flex flex-col gap-4 p-4">
      <SectionCard
        title="Маршрутизация"
        description="Пресеты rule-set для .ru и AI (как в Dahusim). Применяется при connect/reload."
      >
        <SegmentedControl
          options={PRESETS.map((p) => ({
            id: p.id,
            label: p.label,
            hint: p.hint,
          }))}
          value={idx}
          onChange={setIdx}
        />
        <Button
          className="mt-2"
          onClick={async () => {
            await SetRouteQuick(Number(idx));
            setStatus("Маршрут сохранён — применится при connect/reload");
          }}
        >
          Применить
        </Button>
      </SectionCard>
      <Toast text={status} onDismiss={() => setStatus("")} />
    </div>
  );
}
