package logging

import (
	"context"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const modulePrefix = "noxis/"

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
//	noxis/cmd/noxis/server/controller.Deps.submitChange -> Noxis.Controller.Deps.submitChange
//	noxis/internal/services/sql.(*DB).query -> Services.Sql.DB.query
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
	case "services":
		return "Services"
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
