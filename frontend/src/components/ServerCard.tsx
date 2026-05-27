import { cn } from "@/lib/cn";
import { parseServerDisplay, disconnectedServerTitle } from "@/lib/format";

type Props = {
  profileName: string;
  connected: boolean;
  pingText: string;
  pingError?: boolean;
  muted?: boolean;
  onPing?: () => void;
  pingBusy?: boolean;
  pingEnabled?: boolean;
};

export function ServerCard({
  profileName,
  connected,
  pingText,
  pingError,
  muted,
  onPing,
  pingBusy,
  pingEnabled,
}: Props) {
  const display = connected
    ? parseServerDisplay(profileName)
    : disconnectedServerTitle(profileName);

  const pingBadge = (
    <span
      className={cn(
        "shrink-0 rounded-md bg-surface-3 px-2 py-0.5 font-mono text-caption font-semibold",
        pingError ? "text-error" : connected ? "text-success" : "text-muted"
      )}
    >
      {pingBusy ? "…" : pingText}
    </span>
  );

  return (
    <div
      className={cn(
        "card-interactive flex items-start gap-3",
        muted && "opacity-70"
      )}
    >
      <span className="text-[28px] leading-none" aria-hidden>
        {display.flag}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-2">
          <p
            className="truncate text-title text-fg"
            title={display.serverName}
          >
            {display.serverName}
          </p>
          {onPing && pingEnabled ? (
            <button
              type="button"
              className="focus-ring shrink-0"
              disabled={pingBusy}
              onClick={onPing}
              title="Ping"
            >
              {pingBadge}
            </button>
          ) : (
            pingBadge
          )}
        </div>
        {display.showCountry ? (
          <p className="mt-0.5 text-caption text-muted">{display.country}</p>
        ) : null}
      </div>
    </div>
  );
}
