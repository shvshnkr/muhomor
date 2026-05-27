import { useState } from "react";
import { Simple } from "./Simple";
import { ConfigTab } from "../tabs/Config";
import { RouteTab } from "../tabs/Route";
import { SettingsTab } from "../tabs/Settings";
import { SidebarNav } from "@/components/SidebarNav";

type Tab = "simple" | "config" | "route" | "settings";

type Props = { onBack: () => void };

const TABS: { id: Tab; label: string }[] = [
  { id: "simple", label: "Простой" },
  { id: "config", label: "Конфигурация" },
  { id: "route", label: "Маршрут" },
  { id: "settings", label: "Настройки" },
];

export function Extended({ onBack }: Props) {
  const [tab, setTab] = useState<Tab>("simple");

  return (
    <div className="flex h-full min-h-0">
      <SidebarNav items={TABS} active={tab} onSelect={setTab} onBack={onBack} />
      <main className="min-h-0 min-w-0 flex-1 overflow-y-auto bg-bg">
        {tab === "simple" && <Simple embedded />}
        {tab === "config" && <ConfigTab />}
        {tab === "route" && <RouteTab />}
        {tab === "settings" && <SettingsTab />}
      </main>
    </div>
  );
}
