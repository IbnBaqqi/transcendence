import type { HTMLAttributes } from "react";
import Button from "./Button";

interface SkeletonProps extends HTMLAttributes<HTMLDivElement> {
  /** "loading" (default) shows the shimmer. "error" shows a solid block with an optional retry button. */
  variant?: "loading" | "error";
  /** Called when the retry button is pressed. Only relevant when variant="error". */
  onRetry?: () => void;
  /** Short label above the button, e.g. "Couldn't load image". */
  message?: string;
  /** Retry button text. */
  retryLabel?: string;
}

/**
 * Grey placeholder block for loading and error states.
 * Size/shape are controlled entirely via className (w-*, h-*, rounded-*),
 * so this stays a single reusable primitive:
 *
 *   <Skeleton className="h-4 w-48" />                                 // loading, text line
 *   <Skeleton className="h-24 w-24 rounded-full" />                   // loading, avatar
 *   <Skeleton variant="error" className="h-40 w-full" onRetry={fn} /> // error, retry
 */
export function Skeleton({
  variant = "loading",
  onRetry,
  message,
  retryLabel = "Try again",
  className = "",
  ...props
}: SkeletonProps) {
  if (variant === "error") {
    return (
      <div
        className={`bg-surface-soft flex items-center justify-center rounded-md ${className}`}
        {...props}
      >
        <div className="flex flex-col items-center gap-3 p-4 text-center">
          {message && <p className="text-muted text-sm">{message}</p>}
          {onRetry && (
            <Button variant="primary" onClick={onRetry}>
              <div className="inline-flex items-center gap-1.5">
                <RefreshIcon className="size-4" />
                {retryLabel}
              </div>
            </Button>
          )}
        </div>
      </div>
    );
  }

  return <div className={`skeleton rounded-md ${className}`} {...props} />;
}

function RefreshIcon({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="M21 12a9 9 0 1 1-2.64-6.36" />
      <path d="M21 3v6h-6" />
    </svg>
  );
}

/** A few common shapes built on top of Skeleton, for convenience. */
export function SkeletonText({
  lines = 3,
  className = "",
}: {
  lines?: number;
  className?: string;
}) {
  return (
    <div className={`flex flex-col gap-2 ${className}`}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton key={i} className={`h-3 ${i === lines - 1 ? "w-2/3" : "w-full"}`} />
      ))}
    </div>
  );
}

export function SkeletonAvatar({ className = "" }: { className?: string }) {
  return <Skeleton className={`size-10 rounded-full ${className}`} />;
}

export function SkeletonCard({ className = "" }: { className?: string }) {
  return (
    <div className={`flex flex-col gap-3 ${className}`}>
      <Skeleton className="h-40 w-full rounded-lg" />
      <Skeleton className="h-4 w-3/4" />
      <Skeleton className="h-3 w-1/2" />
    </div>
  );
}
