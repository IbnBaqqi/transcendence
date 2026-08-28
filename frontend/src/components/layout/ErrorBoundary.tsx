import { Component, type ReactNode } from "react";

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
      return (
        <div className="mx-auto max-w-2xl p-9 text-center">
          <h1 className="text-foreground text-2xl font-bold">Something went wrong</h1>
          <p className="text-muted mt-2">{this.props.message ?? "Please refresh the page."}</p>
        </div>
      );
    }
    return this.props.children; // no error -> render normally
  }
}
