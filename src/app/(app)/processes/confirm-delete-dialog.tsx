"use client";

import { useEffect, useId, useRef, useState, useTransition } from "react";
import { toast } from "sonner";
import { AlertTriangle } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Props = {
  /** Dialog title. Defaults to `Delete {name}?`. */
  title?: string;
  /** Process name (or other identifier) being deleted. */
  name: string;
  /** Phrase the user has to type — defaults to `name`. */
  challenge?: string;
  /** Confirm-button label, defaults to "Delete forever". */
  confirmLabel?: string;
  /** What the operation does, displayed in the dialog body. */
  description?: string;
  /** Server action to invoke once the user confirms. */
  action: () => Promise<unknown>;
  /** Controlled open state. */
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Optional toast on success. */
  successMessage?: string;
  /** Optional callback to run after the action succeeds (e.g. router.push). */
  onSuccess?: () => void;
};

/**
 * Two-stage destructive confirmation. The user has to:
 *   1. Open the dialog (controlled by the caller).
 *   2. Type the challenge phrase exactly (defaults to the process name —
 *      the same pattern GitHub uses for repo deletion).
 *   3. Click the confirm button.
 *
 * Cancel / Escape / overlay-click all dismiss safely.
 */
export function ConfirmDeleteDialog({
  title,
  name,
  challenge,
  confirmLabel,
  description,
  action,
  open,
  onOpenChange,
  successMessage,
  onSuccess,
}: Props) {
  const challengeText = challenge ?? name;
  const [typed, setTyped] = useState("");
  const [pending, start] = useTransition();
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      setTyped("");
      const t = setTimeout(() => inputRef.current?.focus(), 80);
      return () => clearTimeout(t);
    }
  }, [open]);

  const matches = typed.trim() === challengeText;

  function submit() {
    if (!matches || pending) return;
    start(async () => {
      try {
        await action();
        toast.success(successMessage ?? `Deleted ${name}`);
        onOpenChange(false);
        onSuccess?.();
      } catch (e) {
        toast.error(`${title ?? "Delete"} failed: ${(e as Error).message}`);
      }
    });
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        onKeyDown={(e) => {
          if (e.key === "Enter" && matches) {
            e.preventDefault();
            submit();
          }
        }}
      >
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-full bg-destructive/10">
              <AlertTriangle className="size-4 text-destructive" />
            </div>
            <DialogTitle>{title ?? `Delete ${name}?`}</DialogTitle>
          </div>
          <DialogDescription>
            {description ??
              "This stops every running instance and removes the spec from the dump. Saved logs are kept on disk, but the daemon will no longer supervise this app."}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-1.5">
          <Label htmlFor={inputId} className="block">
            Type <code className="font-mono">{challengeText}</code> to confirm
          </Label>
          <Input
            ref={inputRef}
            id={inputId}
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            placeholder={challengeText}
            autoComplete="off"
            spellCheck={false}
            className="font-mono"
          />
          <p className="text-xs text-muted-foreground">
            Enter to confirm. Cancel or press Esc to back out.
          </p>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="ghost"
            onClick={() => onOpenChange(false)}
            disabled={pending}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={!matches || pending}
            onClick={submit}
          >
            {pending ? "Working…" : confirmLabel ?? "Delete forever"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
