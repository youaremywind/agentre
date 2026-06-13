import { Plus, X, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

export type GrantedChip = {
  id: string;
  label: string;
  count?: number;
  badge?: string; // already-resolved string (e.g. "需审批")
};

type Props = {
  title: string;
  countLabel?: string;
  chipIcon: LucideIcon;
  chips: GrantedChip[];
  addLabel: string;
  removeLabel: (name: string) => string;
  onRemove: (id: string) => void;
  onAdd: () => void;
  emptyLabel?: string;
  footerNote?: string;
  className?: string;
};

export function GrantedChips(props: Props) {
  const Icon = props.chipIcon;
  return (
    <div className={cn("flex flex-col gap-2", props.className)}>
      <div className="flex items-center gap-1.5">
        <h3 className="font-mono text-2xs font-semibold uppercase tracking-wide text-muted-foreground">
          {props.title}
        </h3>
        <div className="flex-1" />
        {props.countLabel && (
          <span className="font-mono text-2xs text-muted-foreground">
            {props.countLabel}
          </span>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        {props.chips.length === 0 && props.emptyLabel && (
          <span className="text-2xs text-muted-foreground">
            {props.emptyLabel}
          </span>
        )}
        {props.chips.map((chip) => (
          <span
            key={chip.id}
            className="inline-flex items-center gap-1.5 rounded-md border border-border bg-card px-2 py-1"
          >
            <Icon className="size-3 text-primary-text" aria-hidden="true" />
            <span className="font-mono text-2xs font-medium">{chip.label}</span>
            {typeof chip.count === "number" && (
              <span className="rounded bg-secondary px-1 font-mono text-2xs text-muted-foreground">
                {chip.count}
              </span>
            )}
            {chip.badge && (
              <span className="rounded bg-status-waiting-bg px-1 font-mono text-2xs text-status-waiting">
                {chip.badge}
              </span>
            )}
            <button
              type="button"
              aria-label={props.removeLabel(chip.label)}
              onClick={() => props.onRemove(chip.id)}
              className="text-muted-foreground hover:text-foreground"
            >
              <X className="size-3" />
            </button>
          </span>
        ))}
        <button
          type="button"
          aria-label={props.addLabel}
          onClick={props.onAdd}
          className="inline-flex items-center gap-1 rounded-md border border-primary/30 bg-primary-soft px-2 py-1 font-mono text-2xs font-semibold text-primary-text hover:bg-primary-soft/70"
        >
          <Plus className="size-3" aria-hidden="true" />
          {props.addLabel}
        </button>
      </div>

      {props.footerNote && (
        <p className="font-mono text-2xs text-muted-foreground">
          {props.footerNote}
        </p>
      )}
    </div>
  );
}
