import { cn } from "@/lib/cn";

type Props = {
  name: string;
  type?: string;
  delayMs?: number;
  enabled?: boolean;
  selected?: boolean;
  onClick?: () => void;
};

export function ProfileRow({
  name,
  type,
  delayMs,
  enabled = true,
  selected,
  onClick,
}: Props) {
  const delay =
    delayMs && delayMs > 0 ? (
      <span className="shrink-0 rounded bg-surface-3 px-1.5 py-0.5 font-mono text-caption text-success">
        {delayMs} ms
      </span>
    ) : null;

  const inner = (
    <>
      <span className={cn("min-w-0 truncate", !enabled && "text-muted")}>
        {name}
      </span>
      <div className="flex shrink-0 items-center gap-1.5">
        {type ? (
          <span className="rounded bg-surface-3 px-1.5 py-0.5 text-caption text-muted">
            {type}
          </span>
        ) : null}
        {delay}
        {!enabled ? (
          <span className="text-caption text-muted">выкл</span>
        ) : null}
      </div>
    </>
  );

  const cls = cn(
    "card-interactive flex w-full items-center justify-between gap-2 px-3 py-2.5 text-left text-sm",
    selected && "border-accent bg-surface-3"
  );

  if (onClick) {
    return (
      <button type="button" className={cn(cls, "focus-ring")} onClick={onClick}>
        {inner}
      </button>
    );
  }
  return <div className={cls}>{inner}</div>;
}
