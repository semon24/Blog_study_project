package blog

import (
	"context"

	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Histogram выбран для анализа распределения времени получения лайков из Redis:
// он накапливает длительности выполнения pipeline, позволяет рассчитывать
// перцентили и долю операций, уложившихся в заданный порог времени.
// В отличие от Gauge, сохраняющего последнее значение, Histogram учитывает
// все измерения; его buckets можно агрегировать между экземплярами приложения.
// Границы заданы в секундах и охватывают быстрые операции Redis, включая 50 мс.
var redisLikesDuration = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name: "blog_redis_likes_duration_seconds",
		Help: "Duration of fetching likes from Redis in seconds.",
		Buckets: []float64{
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1,
		},
	},
)

// Counter выбран для подсчёта завершённых запросов.
var grpcRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "blog_grpc_requests_total",
		Help: "Total number of completed gRPC requests.",
	},
	[]string{"method", "code"},
)

// Histogram выбран для анализа распределения длительности запросов.
var grpcRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "blog_grpc_request_duration_seconds",
		Help: "Duration of gRPC requests in seconds.",
		Buckets: []float64{
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5,
		},
	},
	[]string{"method", "code"},
)

func MetricsInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()

	response, err := handler(ctx, req)

	duration := time.Since(start).Seconds()
	code := status.Code(err).String()

	grpcRequestsTotal.WithLabelValues(
		info.FullMethod,
		status.Code(err).String(),
	).Inc()

	grpcRequestDuration.WithLabelValues(
		info.FullMethod,
		code,
	).Observe(duration)

	return response, err
}
