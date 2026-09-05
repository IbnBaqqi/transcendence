import { useEffect, type RefObject } from "react";

/**
 * Closes an open panel on an outside pointerdown or Escape, and on Escape
 * hands focus back to the trigger - the row the keyboard was standing on has
 * just unmounted, and without this focus falls to the top of the document.
 *
 * Takes the state setter rather than a close callback because useState
 * setters are stable: the listeners re-subscribe once per open, not once per
 * render.
 */
export function useDismissable(
  open: boolean,
  setOpen: (open: boolean) => void,
  rootRef: RefObject<HTMLElement | null>,
  triggerRef?: RefObject<HTMLElement | null>,
) {
  useEffect(() => {
    if (!open) return;

    const onPointerDown = (event: PointerEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) setOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setOpen(false);
      triggerRef?.current?.focus();
    };

    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open, setOpen, rootRef, triggerRef]);
}
