// Avatar initials from a user's real names.
//
// Assumes a signed-in user: the backend guarantees a non-empty username, so
// there is deliberately no "?" fallback here - call sites render "?" based on
// signed-out state instead.

// First character of a string, multibyte safe; whitespace-only counts as unset.
function firstChar(value?: string | null): string {
  const trimmed = value?.trim() ?? "";
  return Array.from(trimmed)[0] ?? "";
}

// NOTE: Overbuilt logic here, currently only using first letter of username
export function deriveInitials(
  username: string,
  firstname?: string | null,
  lastname?: string | null,
): string {
  const first = firstChar(firstname);
  const last = firstChar(lastname);

  if (first && last) return (first + last).toUpperCase();
  if (first) return first.toUpperCase();
  return firstChar(username).toUpperCase();
}
