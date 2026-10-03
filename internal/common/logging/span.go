package logging

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// StartSpan opens a span for one operation (a no-op tracer without a
// configured collector).
func StartSpan(ctx context.Context, name string) (ctxOut context.Context, end func(error)) {
	ctxOut, span := otel.Tracer("noxis").Start(ctx, name)
	return ctxOut, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}
