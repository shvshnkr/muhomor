import { ReactNode, useState } from "react";

import { cn } from "@/lib/cn";



type Props = {

  title: string;

  children: ReactNode;

  defaultOpen?: boolean;

};



export function Collapsible({ title, children, defaultOpen = false }: Props) {

  const [open, setOpen] = useState(defaultOpen);

  return (

    <div className="rounded-card border bg-surface-2/50 shadow-card-inset" style={{ borderColor: "var(--border-subtle)" }}>

      <button

        type="button"

        className="focus-ring flex w-full items-center gap-2 px-3 py-2.5 text-left text-sm font-medium text-muted transition-colors hover:text-fg"

        aria-expanded={open}

        onClick={() => setOpen((v) => !v)}

      >

        <span className={cn("transition-transform", open && "rotate-90")}>

          ▸

        </span>

        {title}

      </button>

      {open ? (

        <div

          className="border-t px-3 py-2"

          style={{ borderColor: "var(--border-subtle)" }}

        >

          {children}

        </div>

      ) : null}

    </div>

  );

}

