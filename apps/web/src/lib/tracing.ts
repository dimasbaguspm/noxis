// Console tracing on the official OpenTelemetry JS SDK. The browser watches a
// navigation, the actions inside it and the API calls they cause, so one trace
// joins the console to the API. With no collector configured the global
// provider stays the noop one: every helper here is a no-op and call sites need
// no enabled check.

import {
  context,
  SpanStatusCode,
  trace,
  type Attributes,
  type Context,
  type Span,
} from "@opentelemetry/api";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-http";
import { resourceFromAttributes } from "@opentelemetry/resources";
import {
  BatchSpanProcessor,
  type ReadableSpan,
  type Span as SdkSpan,
  type SpanProcessor,
} from "@opentelemetry/sdk-trace-base";
import { WebTracerProvider } from "@opentelemetry/sdk-trace-web";
import { ATTR_SERVICE_NAME } from "@opentelemetry/semantic-conventions";

const SERVICE = "noxis-console";

const tracer = trace.getTracer(SERVICE);
let currentParent: Span | undefined;
let currentUser: string | undefined;
let started = false;

// startSpan opens a span under the current page or action, or a new root.
export function startSpan(name: string, attributes?: Attributes): Span {
  const parent = currentParent ? trace.setSpan(context.active(), currentParent) : context.active();
  return tracer.startSpan(name, { attributes }, parent);
}

// setPage makes span the parent of the actions and calls a page makes.
export function setPage(span: Span | undefined): void {
  currentParent = span;
}

// trackAction wraps a handler in an action span. The span stays the parent of
// any API call the handler makes, including after an await: the module keeps it
// current for the handler's duration. shortcut: one action at a time; a second
// overlapping action would take over the parent.
export function trackAction<T>(name: string, fn: () => T | Promise<T>): T | Promise<T> {
  const span = startSpan(`action ${name}`);
  const previous = currentParent;
  currentParent = span;
  const finish = () => {
    currentParent = previous;
    span.end();
  };
  try {
    const result = fn();
    if (result instanceof Promise) {
      return result
        .catch((err) => {
          record(span, err);
          throw err;
        })
        .finally(finish);
    }
    finish();
    return result;
  } catch (err) {
    record(span, err);
    finish();
    throw err;
  }
}

// endSpan ends a span, recording err when there is one.
export function endSpan(span: Span, err?: unknown): void {
  if (err !== undefined) record(span, err);
  span.end();
}

// setUser is the subject stamped on every span as enduser.id.
export function setUser(subject: string | undefined): void {
  currentUser = subject;
}

// beginTracing registers the SDK against the collector. An empty endpoint (the
// API withholds anything that is not https) leaves the noop provider in place.
export function beginTracing(endpoint: string | undefined): void {
  if (started || !endpoint) return;
  started = true;
  const exporter = new OTLPTraceExporter({ url: `${endpoint}/v1/traces` });
  const provider = new WebTracerProvider({
    resource: resourceFromAttributes({ [ATTR_SERVICE_NAME]: SERVICE }),
    spanProcessors: [new UserProcessor(new BatchSpanProcessor(exporter))],
  });
  provider.register();
}

function record(span: Span, err: unknown): void {
  const error = err instanceof Error ? err : new Error(String(err));
  span.recordException(error);
  span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
}

// UserProcessor stamps the signed-in subject on every span before batching. The
// API keeps its own user_id from the bearer token: a browser value is never
// trusted as the source of identity.
class UserProcessor implements SpanProcessor {
  private readonly next: SpanProcessor;

  constructor(next: SpanProcessor) {
    this.next = next;
  }

  onStart(span: SdkSpan, parentContext: Context): void {
    if (currentUser) span.setAttribute("enduser.id", currentUser);
    this.next.onStart(span, parentContext);
  }

  onEnd(span: ReadableSpan): void {
    this.next.onEnd(span);
  }

  forceFlush(): Promise<void> {
    return this.next.forceFlush();
  }

  shutdown(): Promise<void> {
    return this.next.shutdown();
  }
}
