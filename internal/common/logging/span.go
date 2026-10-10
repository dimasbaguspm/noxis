package logging

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const modulePrefix = "noxis/"

// Attach sets attributes on the current span from key/value pairs (string,
// bool and []string values; other types render with fmt). It is a no-op when
// there is no recording span, so callers (the middleware layer) set request
// attributes without caring whether tracing is enabled.
func Attach(ctx context.Context, keyValues ...any) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() || len(keyValues) == 0 {
		return
	}
	attrs := make([]attribute.KeyValue, 0, len(keyValues)/2)
	for i := 0; i+1 < len(keyValues); i += 2 {
		key := attribute.Key(fmt.Sprint(keyValues[i]))
		switch v := keyValues[i+1].(type) {
		case bool:
			attrs = append(attrs, key.Bool(v))
		case []string:
			attrs = append(attrs, key.StringSlice(v))
		default:
			attrs = append(attrs, key.String(fmt.Sprint(v)))
		}
	}
	span.SetAttributes(attrs...)
}

// StartSpan opens a span for the calling function. Name it explicitly, or let
// it derive the project convention: <Module>.<Package>.<Type>.<Method>
// (Repository.Sqlite.Store.List, Services.Sql.DB.query, …).
func StartSpan(ctx context.Context, name ...string) (ctxOut context.Context, end func(error)) {
	spanName := CallerName(1)
	if len(name) > 0 && name[0] != "" {
		spanName = name[0]
	}
	ctxOut, span := otel.Tracer("noxis").Start(ctx, spanName)
	return ctxOut, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

// CallerName is the span name of the function skip frames above the caller of
// CallerName (1 = the caller's own name).
func CallerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	return formatSpanName(runtime.FuncForPC(pc).Name())
}

// formatSpanName turns a runtime function name into the project convention:
//
//	noxis/internal/repositories/sqlite.(*Store).List -> Repository.Sqlite.Store.List
//	noxis/internal/usecases.(*App).SubmitChange -> Usecases.App.SubmitChange
//	noxis/cmd/noxis/server/api.(*Server).handle.func1 -> Noxis.Api.Server.handle.func1
//	noxis/internal/modules/sql.(*DB).query -> Modules.Sql.DB.query
func formatSpanName(full string) string {
	full = strings.TrimPrefix(full, modulePrefix)
	slash := strings.LastIndex(full, "/")
	dot := strings.Index(full[slash+1:], ".")
	if dot < 0 {
		return full
	}
	path := full[:slash+1+dot]
	fn := full[slash+1+dot+1:]
	return spanPrefix(path) + funcName(fn)
}

func spanPrefix(path string) string {
	segs := strings.Split(path, "/")
	switch {
	case len(segs) >= 3 && segs[0] == "internal":
		return moduleName(segs[1]) + "." + title(segs[len(segs)-1]) + "."
	case segs[0] == "cmd" && len(segs) >= 3:
		return "Noxis." + title(segs[len(segs)-1]) + "."
	case segs[0] == "cmd":
		return "Noxis."
	default:
		return title(segs[len(segs)-1]) + "."
	}
}

func moduleName(seg string) string {
	switch seg {
	case "repositories":
		return "Repository"
	case "common":
		return "Common"
	}
	return title(seg)
}

// funcName drops the receiver parens: "(*Store).List" -> "Store.List".
func funcName(fn string) string {
	if !strings.HasPrefix(fn, "(") {
		return fn
	}
	end := strings.Index(fn, ").")
	if end < 0 {
		return fn
	}
	return strings.TrimPrefix(fn[1:end], "*") + "." + fn[end+2:]
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
