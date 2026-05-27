import { cn } from "@/lib/cn";

export type SegmentOption<T extends string> = {
  id: T;
  label: string;
  hint?: string;
};

type Props<T extends string> = {
  options: SegmentOption<T>[];
  value: T;
  onChange: (v: T) => void;
  className?: string;
};

export function SegmentedControl<T extends string>({
  options,
  value,
  onChange,
  className,
}: Props<T>) {
  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div
        className="inline-flex rounded-md border border-border bg-surface-2 p-0.5"
        role="tablist"
      >
        {options.map((o) => (
          <button
            key={o.id}
            type="button"
            role="tab"
            aria-selected={value === o.id}
            className={cn(
              "focus-ring min-h-[44px] flex-1 rounded px-3 py-2 text-sm font-medium transition-colors",
              value === o.id
                ? "bg-accent text-bg"
                : "text-muted hover:text-fg"
            )}
            onClick={() => onChange(o.id)}
          >
            {o.label}
          </button>
        ))}
      </div>
      {options.map(
        (o) =>
          o.hint &&
          value === o.id && (
            <p key={o.id + "-hint"} className="text-caption text-muted">
              {o.hint}
            </p>
          )
      )}
    </div>
  );
}
