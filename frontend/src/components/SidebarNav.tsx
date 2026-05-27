import { cn } from "@/lib/cn";

export type NavItem<T extends string> = { id: T; label: string };

type Props<T extends string> = {
  items: NavItem<T>[];
  active: T;
  onSelect: (id: T) => void;
  onBack?: () => void;
  backLabel?: string;
};

export function SidebarNav<T extends string>({
  items,
  active,
  onSelect,
  onBack,
  backLabel = "← Простой режим",
}: Props<T>) {
  return (
    <nav className="flex w-[200px] shrink-0 flex-col border-r border-border bg-raised py-3">
      {onBack ? (
        <button
          type="button"
          className="focus-ring mx-2 mb-3 px-2 py-1.5 text-left text-sm text-accent hover:text-accent-hover"
          onClick={onBack}
        >
          {backLabel}
        </button>
      ) : null}
      <ul className="flex flex-col gap-0.5 px-2">
        {items.map((item) => {
          const isActive = item.id === active;
          return (
            <li key={item.id}>
              <button
                type="button"
                className={cn(
                  "focus-ring relative w-full rounded-md px-3 py-2.5 text-left text-sm transition-colors",
                  isActive
                    ? "bg-surface-3 text-fg"
                    : "text-muted hover:bg-surface-2 hover:text-fg"
                )}
                onClick={() => onSelect(item.id)}
              >
                {isActive ? (
                  <span
                    className="absolute bottom-2 left-0 top-2 w-0.5 rounded-full bg-accent"
                    aria-hidden
                  />
                ) : null}
                {item.label}
              </button>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
