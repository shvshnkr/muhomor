import { cn } from "@/lib/cn";
import { ButtonHTMLAttributes } from "react";

type Variant = "primary" | "secondary" | "danger";

export function Button({
  variant = "primary",
  className,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  const v =
    variant === "danger"
      ? "btn-danger w-full"
      : variant === "secondary"
        ? "btn-secondary"
        : "btn-primary w-full";
  return <button className={cn(v, className)} {...props} />;
}
