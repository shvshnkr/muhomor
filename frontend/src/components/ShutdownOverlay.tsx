import { cn } from "@/lib/cn";

export type ShutdownState = {
  active: boolean;
  phase?: string;
  message: string;
};

type Props = {
  state: ShutdownState | null;
};

export function ShutdownOverlay({ state }: Props) {
  if (!state?.active) return null;

  const message = state.message?.trim() || "Завершение…";

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-bg"
      role="alertdialog"
      aria-modal="true"
      aria-busy="true"
      aria-label={message}
    >
      <div className="hero-surface flex w-[min(100%,280px)] flex-col items-center gap-4 rounded-2xl border border-white/[0.08] px-6 py-8 text-center shadow-card">
        <div
          className={cn(
            "relative flex h-[72px] w-[72px] items-center justify-center rounded-full",
            "border border-accent/35 bg-accent/10",
            "shadow-power-accent animate-[pulse_1.4s_ease-in-out_infinite]"
          )}
        >
          <span className="h-3 w-3 rounded-full bg-accent shadow-[0_0_12px_var(--accent)]" />
          <span
            className="pointer-events-none absolute inset-0 rounded-full border border-accent/25 animate-ping"
            style={{ animationDuration: "2s" }}
          />
        </div>
        <div className="space-y-1">
          <p className="text-body font-semibold text-fg">{message}</p>
          <p className="text-caption text-muted">
            {state.phase === "daemon"
              ? "Подождите, это может занять до 15 с"
              : "Не закрывайте окно повторно"}
          </p>
        </div>
      </div>
    </div>
  );
}
