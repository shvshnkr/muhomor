import { cn } from "@/lib/cn";



type Variant = "idle" | "connecting" | "connected" | "error";



const VARIANT: Record<Variant, { dot: string; text: string }> = {

  idle: { dot: "bg-muted", text: "text-muted" },

  connecting: { dot: "bg-warn animate-pulse", text: "text-warn" },

  connected: { dot: "bg-success ring-2 ring-accent/40", text: "text-success" },

  error: { dot: "bg-error", text: "text-error" },

};



type Props = {

  label: string;

  variant: Variant;

  activity?: string;

};

function showActivity(label: string, activity?: string): string | undefined {
  const a = activity?.trim();
  if (!a) return undefined;
  if (a === label.trim()) return undefined;
  if (a.startsWith("Подключение") && label.trim().startsWith("Подключение")) {
    return undefined;
  }
  return a;
}

export function StatusPill({ label, variant, activity }: Props) {

  const v = VARIANT[variant];
  const activityLine = showActivity(label, activity);

  return (

    <div className="flex flex-col items-center gap-1 text-center">

      <div

        className={cn(

          "inline-flex items-center gap-2 rounded-full border border-white/[0.08] bg-surface-2/60 px-3 py-1.5 backdrop-blur-sm transition-opacity",

          v.text

        )}

      >

        <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", v.dot)} />

        <span className="text-body font-semibold">{label}</span>

      </div>

      {activityLine ? (

        <p

          className={cn(

            "max-w-full truncate px-2 text-caption",

            variant === "error" ? "text-error/90" : "text-muted"

          )}

        >

          {activityLine}

        </p>

      ) : null}

    </div>

  );

}

