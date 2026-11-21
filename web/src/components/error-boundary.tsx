import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { Alert } from './alert';

// A console that renders nothing when a screen throws is a console nobody can debug: the operator sees
// a white page and has nothing to send anyone. This is the last resort — the message, the component
// stack, and a reload — and its whole job is to make the failure legible instead of empty.

type State = { error: Error | null; stack: string };

export class ErrorBoundary extends Component<{ children: ReactNode }, State> {
  state: State = { error: null, stack: '' };

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // The browser console is where a stack belongs; shipping one to the API would be a telemetry
    // decision nobody made.
    this.setState({ stack: info.componentStack ?? '' });
    console.error('a screen failed to render', error, info.componentStack);
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="p-4 md:p-6">
        <Alert
          tone="error"
          title="This screen failed to render"
          description={
            <div className="flex flex-col gap-2">
              <span>{this.state.error.message}</span>
              <details className="text-xs">
                <summary className="cursor-pointer text-muted">the component stack</summary>
                <pre className="mt-2 max-h-48 overflow-auto text-[11px] whitespace-pre-wrap">
                  {this.state.stack}
                </pre>
              </details>
            </div>
          }
          action={
            <button
              type="button"
              className="text-sm font-semibold text-brand-ink underline"
              onClick={() => window.location.reload()}
            >
              Reload
            </button>
          }
        />
      </div>
    );
  }
}
