import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { ProfileRow } from "@/components/ProfileRow";
import { SectionCard } from "@/components/SectionCard";
import { Toast } from "@/components/Toast";
import {
  AddGroupServer,
  GroupDTO,
  ListGroups,
  ListProfiles,
  ProfileDTO,
  RefreshGroup,
  UpdateGroupSubscription,
} from "../wailsjs/go/main/App";

export function ConfigTab() {
  const [groups, setGroups] = useState<GroupDTO[]>([]);
  const [profiles, setProfiles] = useState<ProfileDTO[]>([]);
  const [selGroup, setSelGroup] = useState(0);
  const [subLink, setSubLink] = useState("");
  const [serverURI, setServerURI] = useState("");
  const [status, setStatus] = useState("");
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);

  const g = groups[selGroup];
  const q = filter.trim().toLowerCase();
  const filtered = profiles.filter((p) => {
    if (g && p.groupId !== g.id) return false;
    if (q && !p.name.toLowerCase().includes(q)) return false;
    return true;
  });

  const isSubscription = g?.kind === "subscription";

  useEffect(() => {
    setLoading(true);
    Promise.all([ListGroups(), ListProfiles()])
      .then(([gr, pr]) => {
        setGroups(gr);
        setProfiles(pr);
      })
      .catch((e) => setStatus(String(e)))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (g) setSubLink(g.subscriptionLink || "");
  }, [g?.id]);

  return (
    <div className="flex h-full min-h-0 flex-col gap-3 p-4">
      <div className="grid min-h-0 flex-1 grid-cols-[28%_1fr] gap-3">
        <div className="card flex min-h-0 flex-col overflow-hidden">
          <p className="mb-2 text-title text-fg">Группы</p>
          {loading ? (
            <p className="text-caption text-muted">Загрузка…</p>
          ) : (
            <ul className="min-h-0 flex-1 overflow-y-auto">
              {groups.map((gr, i) => (
                <li key={gr.id}>
                  <button
                    type="button"
                    className={`focus-ring w-full rounded px-2 py-2 text-left text-sm hover:bg-surface-3 ${selGroup === i ? "bg-surface-3 text-accent" : ""}`}
                    onClick={() => setSelGroup(i)}
                  >
                    {gr.label}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="card flex min-h-0 flex-col overflow-hidden">
          <input
            className="focus-ring mb-2 rounded border border-border bg-surface-2 px-2 py-2 text-sm"
            placeholder="Поиск по имени…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
          <ul className="min-h-0 flex-1 space-y-1 overflow-y-auto">
            {filtered.length === 0 ? (
              <li className="py-4 text-center text-caption text-muted">
                {loading ? "Загрузка…" : "Нет серверов в группе"}
              </li>
            ) : (
              filtered.slice(0, 300).map((p) => (
                <li key={p.id}>
                  <ProfileRow
                    name={p.name}
                    type={p.type}
                    delayMs={p.lastDelayMs}
                    enabled={p.enabled}
                  />
                </li>
              ))
            )}
          </ul>
        </div>
      </div>

      {g ? (
        <SectionCard
          title={isSubscription ? "Подписка" : "Ручная группа"}
          description={
            isSubscription
              ? "URL подписки и обновление списка серверов"
              : "Добавление сервера по URI"
          }
        >
          {isSubscription ? (
            <>
              <label className="text-caption text-muted">URL подписки</label>
              <input
                className="focus-ring w-full rounded border border-border bg-surface-2 px-2 py-2 text-sm"
                value={subLink}
                onChange={(e) => setSubLink(e.target.value)}
              />
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="secondary"
                  className="!min-h-9 !w-auto text-sm"
                  onClick={async () => {
                    await UpdateGroupSubscription(g.id, subLink);
                    setStatus("Сохранено");
                  }}
                >
                  Сохранить URL
                </Button>
                <Button
                  variant="primary"
                  className="!min-h-9 !w-auto text-sm"
                  onClick={async () => {
                    const msg = await RefreshGroup(g.id);
                    setStatus(msg);
                    ListProfiles().then(setProfiles);
                  }}
                >
                  Обновить подписку
                </Button>
              </div>
            </>
          ) : (
            <>
              <input
                className="focus-ring w-full rounded border border-border bg-surface-2 px-2 py-2 text-sm"
                placeholder="vless://…"
                value={serverURI}
                onChange={(e) => setServerURI(e.target.value)}
              />
              <Button
                variant="primary"
                className="!min-h-9 !w-auto text-sm"
                onClick={async () => {
                  await AddGroupServer(g.id, serverURI);
                  setStatus("Сервер добавлен");
                  ListProfiles().then(setProfiles);
                }}
              >
                Добавить сервер
              </Button>
            </>
          )}
        </SectionCard>
      ) : null}

      <Toast text={status} onDismiss={() => setStatus("")} />
    </div>
  );
}
