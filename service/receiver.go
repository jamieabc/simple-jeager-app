package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

const (
	serviceName = "receiver"
	servicePort = ":7777"
)

func main() {
	tracer, closer := initJaeger(serviceName)
	defer closer.Close()

	http.HandleFunc("/receiver", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, err := tracer.Extract(
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(r.Header),
		)

		if nil != err {
			fmt.Printf("Error: cannot extract span context: %v\n", err)
		}

		span := tracer.StartSpan("receiver", ext.RPCServerOption(spanCtx))
		defer span.Finish()

		blockNumber := span.BaggageItem("block-number")

		if "" == blockNumber {
			blockNumber = "0"
		}

		span.LogFields(
			otlog.String("event", "receiver"),
			otlog.String("value", blockNumber),
		)
		str := fmt.Sprintf("block number: %s", blockNumber)
		w.Write([]byte(str))
	})

	log.Fatal(http.ListenAndServe(servicePort, nil))
}

func initJaeger(serviceName string) (opentracing.Tracer, io.Closer) {
	cfg := &jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
		},
	}

	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	tracer, closer, err := cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)

	if nil != err {
		panic(fmt.Errorf("Error: cannot init Jaeger: %v\n", err))
	}

	return tracer, closer
}
