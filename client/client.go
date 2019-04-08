package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

const (
	serviceName = "client"
	spanName    = "api-call"
	servicePort = ":7777"
	serviceURN  = "receiver"
)

func main() {
	tracer, closer := initJaeger(serviceName)
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	rootSpan := tracer.StartSpan(spanName)
	defer rootSpan.Finish()

	blockNumber := strconv.FormatInt(time.Now().Unix(), 10)
	rootSpan.SetTag("source", "root")
	rootSpan.SetBaggageItem("block-number", blockNumber)

	message := "hello world"
	ctx := opentracing.ContextWithSpan(context.Background(), rootSpan)
	sendToService(ctx, message)
}

func sendToService(ctx context.Context, message string) {
	span, _ := opentracing.StartSpanFromContext(ctx, "send to service")
	defer span.Finish()

	v := url.Values{}
	v.Set("msg", message)
	url := fmt.Sprintf("http://localhost%s/%s?%s", servicePort, serviceURN, v.Encode())
	req, err := http.NewRequest("GET", url, nil)
	if nil != err {
		panic(err.Error())
	}

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "GET")
	span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)

	httpClient := &http.Client{}

	resp, err := httpClient.Do(req)
	if nil != err {
		panic(err.Error())
	}
	defer resp.Body.Close()

	if nil != err {
		panic(err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		panic(err.Error)
	}
	fmt.Printf("response: %s\n", body)

	span.LogFields(
		otlog.String("event", "send to service"),
		otlog.String("value", string(body)),
	)
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
