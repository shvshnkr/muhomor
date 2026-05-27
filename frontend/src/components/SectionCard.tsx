import { ReactNode } from "react";

type Props = {
  title: string;
  description?: string;
  children: ReactNode;
};

export function SectionCard({ title, description, children }: Props) {
  return (
    <section className="card space-y-3">
      <div>
        <h2 className="text-title text-fg">{title}</h2>
        {description ? (
          <p className="mt-1 text-caption text-muted">{description}</p>
        ) : null}
      </div>
      {children}
    </section>
  );
}
