// The console's trace context. It is deliberately tiny: the browser only needs
// to originate a W3C trace and hand its id to the API, so one trace covers a
// navigation and the backend calls it causes. Spans stay in memory; shipping
// them to the collector is a separate concern.

export interface Span {
  traceId: string;
  spanId: string;
  name: string;
  startTime: number;
  endTime?: number;
  error?: string;
}

// shortcut: the buffer keeps the last 200 spans for the session, an export
// flushes and trimming replaces the cap.
const MAX_SPANS = 200;
const spans: Span[] = [];
let activeTraceId: string | undefined;

function randomHex(bytes: number): string {
  const buf = new Uint8Array(bytes);
  crypto.getRandomValues(buf);
  return Array.from(buf, (b) => b.toString(16).padStart(2, "0")).join("");
}

// beginTrace starts a trace and makes it the parent of later spans.
export function beginTrace(): string {
  activeTraceId = randomHex(16);
  return activeTraceId;
}

// endTrace clears the active trace once its navigation unmounts.
export function endTrace(traceId: string): void {
  if (activeTraceId === traceId) activeTraceId = undefined;
}

// startSpan opens a span under an explicit trace, the active one, or a new one.
export function startSpan(name: string, traceId?: string): Span {
  const span: Span = {
    traceId: traceId ?? activeTraceId ?? randomHex(16),
    spanId: randomHex(8),
    name,
    startTime: Date.now(),
  };
  spans.push(span);
  if (spans.length > MAX_SPANS) spans.shift();
  return span;
}

export function endSpan(span: Span, err?: unknown): void {
  if (span.endTime !== undefined) return;
  span.endTime = Date.now();
  if (err) span.error = err instanceof Error ? err.message : String(err);
}

// traceparent is the W3C header a request carries so the API joins this trace.
export function traceparent(span: Span): string {
  return `00-${span.traceId}-${span.spanId}-01`;
}

// finishedSpans returns the ended spans, oldest first.
export function finishedSpans(): Span[] {
  return spans.filter((span) => span.endTime !== undefined);
}
