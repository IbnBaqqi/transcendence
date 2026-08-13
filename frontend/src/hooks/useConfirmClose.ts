export function useConfirmClose(isDirty: boolean, onClose: () => void) {
  return () => {
    if (isDirty && !window.confirm("Discard unsaved changed?")) return;
    onClose();
  };
}
