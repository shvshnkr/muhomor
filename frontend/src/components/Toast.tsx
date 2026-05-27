import { cn } from "@/lib/cn";



type Props = {

  text: string;

  error?: boolean;

  onDismiss?: () => void;

};



export function Toast({ text, error, onDismiss }: Props) {

  if (!text) return null;

  return (

    <div

      role={error ? "alert" : "status"}

      className={cn(

        "flex items-start gap-3 rounded-xl border px-4 py-3 text-caption transition-colors",

        error

          ? "border-error/25 bg-error/[0.08] text-error shadow-[0_4px_20px_rgba(248,113,113,0.08)]"

          : "border-warn/25 bg-warn/[0.08] text-warn shadow-[0_4px_20px_rgba(245,158,11,0.06)]",

        onDismiss && "cursor-pointer hover:brightness-110"

      )}

      onClick={onDismiss}

    >

      <span

        className={cn(

          "mt-1.5 h-2 w-2 shrink-0 rounded-full",

          error ? "bg-error" : "bg-warn"

        )}

        aria-hidden

      />

      <p className="min-w-0 flex-1 leading-snug">{text}</p>

      {onDismiss ? (

        <button

          type="button"

          className="focus-ring wails-no-drag shrink-0 rounded px-1 text-muted hover:text-fg"

          aria-label="Закрыть"

          onClick={(e) => {

            e.stopPropagation();

            onDismiss();

          }}

        >

          ×

        </button>

      ) : null}

    </div>

  );

}

