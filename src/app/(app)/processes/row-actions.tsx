"use client";

import { useState, useTransition } from "react";
import { toast } from "sonner";
import {
  RefreshCw,
  RotateCw,
  Square,
  Trash2,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  deleteAction,
  reloadAction,
  restartAction,
  stopAction,
} from "./actions";
import { ConfirmDeleteDialog } from "./confirm-delete-dialog";

type Props = {
  name: string;
  state: string;
  canWrite: boolean;
  canDelete: boolean;
};

/**
 * Inline icon-button row instead of a dropdown — one click per action,
 * destructive ops route through ConfirmDeleteDialog with a re-type challenge.
 */
export function ProcessRowActions({ name, state, canWrite, canDelete }: Props) {
  const [pending, start] = useTransition();
  const [confirmOpen, setConfirmOpen] = useState(false);
  if (!canWrite && !canDelete) return null;

  const run = (label: string, fn: () => Promise<unknown>) =>
    start(async () => {
      try {
        await fn();
        toast.success(`${label}: ${name}`);
      } catch (e) {
        toast.error(`${label} ${name} failed: ${(e as Error).message}`);
      }
    });

  return (
    <>
      <div className="flex items-center justify-end gap-1">
        {canWrite ? (
          <>
            <IconButton
              label="Restart"
              icon={RotateCw}
              disabled={pending}
              onClick={() => run("Restarted", () => restartAction(name))}
            />
            <IconButton
              label="Reload (graceful, SIGUSR2)"
              icon={RefreshCw}
              disabled={pending}
              onClick={() => run("Reloaded", () => reloadAction(name))}
            />
            <IconButton
              label="Stop"
              icon={Square}
              disabled={pending || state === "stopped"}
              onClick={() => run("Stopped", () => stopAction(name))}
            />
          </>
        ) : null}
        {canDelete ? (
          <IconButton
            label="Delete"
            icon={Trash2}
            danger
            disabled={pending}
            onClick={() => setConfirmOpen(true)}
          />
        ) : null}
      </div>
      {canDelete ? (
        <ConfirmDeleteDialog
          name={name}
          action={() => deleteAction(name)}
          open={confirmOpen}
          onOpenChange={setConfirmOpen}
        />
      ) : null}
    </>
  );
}

function IconButton({
  label,
  icon: Icon,
  onClick,
  disabled,
  danger,
}: {
  label: string;
  icon: LucideIcon;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
}) {
  return (
    <Button
      type="button"
      size="icon"
      variant="ghost"
      title={label}
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      className={
        danger ? "text-red-600 hover:bg-red-500/10 dark:text-red-400" : undefined
      }
    >
      <Icon className="h-4 w-4" />
    </Button>
  );
}
