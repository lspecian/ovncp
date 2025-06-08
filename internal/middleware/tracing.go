package middleware

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/lspecian/ovncp/internal/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracing middleware for distributed tracing
func Tracing(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// Extract trace context from headers
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start span
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(c.Request.Method),
				semconv.HTTPTargetKey.String(c.Request.URL.Path),
				semconv.HTTPRouteKey.String(c.FullPath()),
				semconv.HTTPURLKey.String(c.Request.URL.String()),
				semconv.HTTPSchemeKey.String(c.Request.URL.Scheme),
				semconv.HTTPHostKey.String(c.Request.Host),
				semconv.HTTPUserAgentKey.String(c.Request.UserAgent()),
				semconv.HTTPRequestContentLengthKey.Int64(c.Request.ContentLength),
				semconv.NetPeerIPKey.String(c.ClientIP()),
			),
		)
		defer span.End()

		// Store span in context for later use
		c.Request = c.Request.WithContext(ctx)
		c.Set("span", span)

		// Process request
		c.Next()

		// Update span with response info
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(c.Writer.Status()),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Record error if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
			span.SetStatus(trace.Status{
				Code:    trace.StatusCodeError,
				Message: c.Errors.Last().Error(),
			})
		} else if c.Writer.Status() >= 400 {
			span.SetStatus(trace.Status{
				Code:    trace.StatusCodeError,
				Message: fmt.Sprintf("HTTP %d", c.Writer.Status()),
			})
		}

		// Inject trace context into response headers
		propagator.Inject(ctx, propagation.HeaderCarrier(c.Writer.Header()))
	}
}

// GetSpanFromContext gets the current span from gin context
func GetSpanFromContext(c *gin.Context) trace.Span {
	if span, exists := c.Get("span"); exists {
		return span.(trace.Span)
	}
	return trace.SpanFromContext(c.Request.Context())
}

// StartSpanFromContext starts a new span as a child of the current span
func StartSpanFromContext(c *gin.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracing.StartSpan(c.Request.Context(), name, opts...)
}

// AddSpanTags adds tags to the current span
func AddSpanTags(c *gin.Context, tags ...attribute.KeyValue) {
	span := GetSpanFromContext(c)
	span.SetAttributes(tags...)
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(c *gin.Context, name string, attrs ...attribute.KeyValue) {
	span := GetSpanFromContext(c)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordSpanError records an error on the current span
func RecordSpanError(c *gin.Context, err error) {
	span := GetSpanFromContext(c)
	span.RecordError(err)
	span.SetStatus(trace.Status{
		Code:    trace.StatusCodeError,
		Message: err.Error(),
	})
}