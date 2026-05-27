import { cn } from "@/lib/cn";



type State = "idle" | "connecting" | "connected" | "danger";



type Props = {

  label: string;

  state: State;

  disabled?: boolean;

  onClick: () => void;

};



export function ConnectPowerButton({ label, state, disabled, onClick }: Props) {

  return (

    <button

      type="button"

      disabled={disabled}

      onClick={onClick}

      className={cn(

        "focus-ring mx-auto flex h-[132px] w-[132px] flex-col items-center justify-center rounded-full border text-title font-semibold transition-all duration-300 ease-out disabled:opacity-50",

        state === "connected" &&

          "border-accent/40 bg-accent/15 text-accent shadow-power-accent",

        state === "connecting" &&

          "border-warn/40 bg-warn/10 text-warn shadow-power-idle animate-[pulse_1.4s_ease-in-out_infinite]",

        state === "danger" &&

          "border-error/30 bg-error/[0.08] text-error shadow-power-danger hover:border-error/45 hover:bg-error/15 active:shadow-[inset_0_2px_8px_rgba(0,0,0,0.25)]",

        state === "idle" &&

          "border-border-strong bg-surface-2 text-fg shadow-power-idle hover:border-[var(--border-hover)] hover:bg-surface-3 active:shadow-[inset_0_2px_8px_rgba(0,0,0,0.2)]"

      )}

    >

      {label}

    </button>

  );

}

