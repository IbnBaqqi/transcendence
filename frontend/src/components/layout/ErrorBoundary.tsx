import { Component, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

interface Props {
  children: ReactNode; // whatever the boundary wraps
  // What to tell the user. The default suits the boundary above the router in
  // main.tsx: when that one fires the header is gone, so refreshing really is
  // the only way out. A boundary inside the shell passes something better.
  message?: string;
}
interface State {
  hasError: boolean;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  // a child threw during render -> flip to the fallback UI
  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  // recvieves the actual error -> place for side effects like logging
  componentDidCatch(error: unknown) {
    console.error("Uncaught error:", error);
  }

  render() {
    if (this.state.hasError) {
      return <ErrorBoundaryFallback message={this.props.message} />;
    }
    return this.props.children; // no error -> render normally
  }
}

// The boundary itself stays a class (React requires that to catch errors), so
// the hook lives in this small function component instead.
function ErrorBoundaryFallback({ message }: { message?: string }) {
  const { t } = useTranslation();
  return (
    <div className="max-w-column mx-auto p-9 text-center">
      <h1 className="text-foreground text-page-title font-bold">
        {t("errorBoundary.somethingWentWrong")}
      </h1>
      <p className="text-muted mt-2">{message ?? t("errorBoundary.refresh")}</p>
    </div>
  );
}
