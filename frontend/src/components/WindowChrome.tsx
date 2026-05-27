import {
  WindowHide,
  WindowMinimise,
  WindowToggleMaximise,
} from "../wailsjs/runtime/runtime";
import { useShutdown } from "@/stores/shutdown";

type Props = {
  title?: string;
};

function ChromeButton({
  label,
  onClick,
  className,
  disabled,
}: {
  label: string;
  onClick: () => void;
  className?: string;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      disabled={disabled}
      className={`wails-no-drag flex h-7 w-7 items-center justify-center rounded text-sm text-muted transition-colors hover:bg-white/[0.06] hover:text-fg disabled:pointer-events-none disabled:opacity-40 ${className ?? ""}`}
      onClick={onClick}
    >
      {label}
    </button>
  );
}

export function WindowChrome({ title = "muhomor" }: Props) {
  const { shuttingDown } = useShutdown();

  return (
    <header className="wails-drag flex h-8 shrink-0 items-center border-b border-white/[0.06] bg-bg/95 pl-3 pr-1">
      <span className="min-w-0 flex-1 truncate text-caption tracking-wide text-muted">
        {title}
      </span>
      <div className="flex shrink-0 items-center gap-0.5">
        <ChromeButton
          label="−"
          disabled={shuttingDown}
          onClick={() => WindowMinimise()}
        />
        <ChromeButton
          label="□"
          disabled={shuttingDown}
          onClick={() => WindowToggleMaximise()}
        />
        <ChromeButton
          label="×"
          disabled={shuttingDown}
          className="hover:bg-white/[0.08] hover:text-fg"
          onClick={() => WindowHide()}
        />
      </div>
    </header>
  );
}
