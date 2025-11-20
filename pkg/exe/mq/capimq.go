package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/capillariesio/capillaries/pkg/capimq_message_broker"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type UrlHandlerInstance struct {
	Env *env.EnvConfig
	L   *l.CapiLogger
	Mb  *capimq_message_broker.MessageBroker
}

type ctxKey struct {
}

func getFieldByIndexFromRequest(r *http.Request, index int) (string, error) {
	fields, ok := r.Context().Value(ctxKey{}).([]string)
	if !ok {
		return "", fmt.Errorf("cannot obtain request field %d, no fields in http request", index)
	}
	if len(fields) <= index {
		return "", fmt.Errorf("cannot obtain request field %d, only %d fields available", index, len(fields))
	}
	return fields[index], nil
}

func (h UrlHandlerInstance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var allow []string
	for _, route := range routes {
		matches := route.regex.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 {
			if r.Method != route.method {
				allow = append(allow, route.method)
				continue
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, matches[1:])

			route.handler(w, r.WithContext(ctx))
			return
		}
	}
	if len(allow) > 0 {
		w.Header().Set("Allow", strings.Join(allow, ", "))
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.NotFound(w, r)
}

type route struct {
	method  string
	regex   *regexp.Regexp
	handler http.HandlerFunc
}

var routes []route

func newRoute(method, pattern string, handler http.HandlerFunc) route {
	return route{method, regexp.MustCompile("^" + pattern + "$"), handler}
}

func pickAccessControlAllowOrigin(wc *env.CapiMqBrokerConfig, r *http.Request) string {
	if wc.AccessControlAllowOrigin == "*" {
		return "*"
	}
	allowedOrigins := strings.Split(wc.AccessControlAllowOrigin, ",")
	requestedOrigins, ok := r.Header["Origin"]
	if !ok || len(requestedOrigins) == 0 {
		return "no-origins-requested"
	}
	for _, allowedOrigin := range allowedOrigins {
		for _, requestedOrigin := range requestedOrigins {
			if strings.EqualFold(requestedOrigin, allowedOrigin) {
				return requestedOrigin
			}
		}
	}
	return "no-allowed-origins"
}

func WriteApiError(logger *l.CapiLogger, wc *env.CapiMqBrokerConfig, r *http.Request, w http.ResponseWriter, urlPath string, err error, httpStatus int) {
	logger.PushF("WriteApiError")
	defer logger.PopF()

	ApiErrorHitCounter.Inc()

	w.Header().Set("Access-Control-Allow-Origin", pickAccessControlAllowOrigin(wc, r))
	logger.Error("cannot process %s: %s", urlPath, err.Error())
	respJson, err := json.Marshal(capimq_message_broker.CapimqApiGenericResponse{Error: err.Error()})
	if err != nil {
		http.Error(w, fmt.Sprintf("unexpected: cannot serialize error response %s", err.Error()), httpStatus)
	} else {
		http.Error(w, string(respJson), httpStatus)
	}
}

func WriteApiSuccess(logger *l.CapiLogger, wc *env.CapiMqBrokerConfig, r *http.Request, w http.ResponseWriter, data any) {
	logger.PushF("WriteApiSuccess")
	defer logger.PopF()

	ApiSuccessHitCounter.Inc()

	logger.Debug("%s: OK", r.URL.Path)

	w.Header().Set("Access-Control-Allow-Origin", pickAccessControlAllowOrigin(wc, r))
	respJson, err := json.Marshal(capimq_message_broker.CapimqApiGenericResponse{Data: data})
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot serialize success response: %s", err.Error()), http.StatusInternalServerError)
	} else {
		if _, err := w.Write([]byte(respJson)); err != nil {
			logger.Error("cannot write success response, error %s, response %s", err.Error(), respJson)
		}
	}
}

func (h *UrlHandlerInstance) qBulk(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	var msgs []*capimq_message_broker.CapimqMessage
	if err = json.Unmarshal(bodyBytes, &msgs); err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	now := time.Now().UnixMilli()
	internalMsgs := make([]*capimq_message_broker.CapimqInternalMessage, len(msgs))
	for i := range len(msgs) {
		internalMsgs[i] = &capimq_message_broker.CapimqInternalMessage{
			Id:                   msgs[i].Id,
			CapimqWaitRetryGroup: msgs[i].CapimqWaitRetryGroup,
			Ts:                   now,
			Heartbeat:            now,
			DeliverAfter:         now,
			Data:                 msgs[i].Data,
			ClaimComment:         "",
		}
	}

	if err := h.Mb.QBulk(internalMsgs); err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, nil)
}

func (h *UrlHandlerInstance) qClaim(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	claimComment := string(bodyBytes)

	msg, err := h.Mb.Claim(claimComment)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, msg.ToCapimqMessage())
}

func (h *UrlHandlerInstance) wipAck(w http.ResponseWriter, r *http.Request) {
	id, err := getIdFromRequest(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	if err := h.Mb.Ack(id); err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, nil)
}

func (h *UrlHandlerInstance) wipHeartbeat(w http.ResponseWriter, r *http.Request) {
	id, err := getIdFromRequest(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	if err := h.Mb.Heartbeat(id); err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, nil)
}

func (h *UrlHandlerInstance) wipReturn(w http.ResponseWriter, r *http.Request) {
	id, err := getIdFromRequest(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	if err := h.Mb.Return(id, int64(h.Env.CapiMqBroker.ReturnedDeliveryDelay)); err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, nil)
}

func getHeapTypeFromRequest(r *http.Request) (capimq_message_broker.HeapType, error) {
	qOrWip, err := getFieldByIndexFromRequest(r, 0)
	if err != nil {
		return capimq_message_broker.HeapTypeUnknown, err
	}

	return capimq_message_broker.StringToHeapType(qOrWip)
}

func getQueueStartFromRequest(r *http.Request) (capimq_message_broker.QueueReadType, error) {
	headOrTail, err := getFieldByIndexFromRequest(r, 1)
	if err != nil {
		return capimq_message_broker.QueueReadUnknown, err
	}

	return capimq_message_broker.StringToQueueReadType(headOrTail)
}

func getIdFromRequest(r *http.Request, idx int) (string, error) {
	id, err := getFieldByIndexFromRequest(r, idx)
	if err != nil {
		return "", err
	}

	if len(id) == 0 {
		return "", fmt.Errorf("meaningless id parameter %s", id)
	}

	return id, nil
}

func getFromCountParamsFromQuery(r *http.Request) (int, int, error) {
	from_string := r.URL.Query().Get("from")
	count_string := r.URL.Query().Get("count")
	from := int64(0)
	count := int64(0)
	var err error
	if len(from_string) > 0 {
		from, err = strconv.ParseInt(from_string, 10, 64)
		if err != nil || from < 0 {
			return 0, 0, fmt.Errorf("invalid from parameter %s", from_string)
		}
	}
	if len(count_string) > 0 {
		count, err = strconv.ParseInt(count_string, 10, 64)
		if err != nil || count <= 0 {
			return 0, 0, fmt.Errorf("invalid count parameter %s", count_string)
		}
	}
	return int(from), int(count), nil
}

func (h *UrlHandlerInstance) count(w http.ResponseWriter, r *http.Request) {
	heapType, err := getHeapTypeFromRequest(r)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	waitRetryGroupPrefix := r.URL.Query().Get("waitRetryGroupPrefix")
	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, h.Mb.Count(heapType, waitRetryGroupPrefix))
}

func (h *UrlHandlerInstance) delete(w http.ResponseWriter, r *http.Request) {
	heapType, err := getHeapTypeFromRequest(r)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	waitRetryGroupPrefix := r.URL.Query().Get("waitRetryGroupPrefix")
	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, h.Mb.Delete(heapType, waitRetryGroupPrefix))
}

func (h *UrlHandlerInstance) headTailFilter(w http.ResponseWriter, r *http.Request) {
	heapType, err := getHeapTypeFromRequest(r)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	queueRead, err := getQueueStartFromRequest(r)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	switch queueRead {
	case capimq_message_broker.QueueReadHead, capimq_message_broker.QueueReadTail:
		from, count, err := getFromCountParamsFromQuery(r)
		if err != nil {
			WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		}
		WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, capimq_message_broker.ToCapimqMessages(h.Mb.HeadTail(heapType, queueRead, from, count)))
	case capimq_message_broker.QueueReadFilter:
		waitRetryGroupPrefix := r.URL.Query().Get("waitRetryGroupPrefix")
		WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, capimq_message_broker.ToCapimqMessages(h.Mb.Filter(heapType, waitRetryGroupPrefix)))
	default:
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, fmt.Errorf("unexpected queueRead %s", queueRead), http.StatusInternalServerError)
	}
}

var (
	ApiSuccessHitCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_mq_api_success_hit_count",
		Help: "Capillaries CapiMq API success hit count",
	})
	ApiErrorHitCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_mq_api_error_hit_count",
		Help: "Capillaries CapiMq API error hit count",
	})
	ReturnDeadTimeoutCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_mq_return_dead_timeout_count",
		Help: "Capillaries CapiMq count of messages returned from wip back to queue because of timeout",
	})
)

var version string

func main() {
	initCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	envConfig, err := env.ReadEnvConfigFile(initCtx, "capimq.json")
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	logger, err := l.NewLoggerFromEnvConfig(envConfig)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	defer logger.Close()

	logger.Info("Capillaries mq %s", version)
	logger.Info("env config: %s", envConfig.String())

	mux := http.NewServeMux()

	h := UrlHandlerInstance{Env: envConfig, L: logger, Mb: capimq_message_broker.NewMessageBroker(envConfig.CapiMqBroker.MaxMessages)}

	routes = []route{
		newRoute("POST", "/q/bulk[/]*", h.qBulk),
		newRoute("POST", "/q/claim[/]*", h.qClaim),
		newRoute("DELETE", "/wip/ack/([A-Fa-f0-9-]+)[/]*", h.wipAck),
		newRoute("POST", "/wip/heartbeat/([A-Fa-f0-9-]+)[/]*", h.wipHeartbeat),
		newRoute("POST", "/wip/return/([A-Fa-f0-9-]+)[/]*", h.wipReturn),

		newRoute("GET", "/(q|wip)/count[/]*", h.count),
		newRoute("DELETE", "/(q|wip)[/]*", h.delete),
		newRoute("GET", "/(q|wip)/(head|tail|filter)[/]*", h.headTailFilter),
		newRoute("GET", "/(q|wip)/(head|tail|filter)[/]*", h.headTailFilter),
	}

	mux.Handle("/", h)

	waitGroup := &sync.WaitGroup{}

	waitGroup.Add(1)
	returnDeadStopping := false
	returnedDeadStoppingLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	defer returnedDeadStoppingLogger.Close()

	go func(innerLogger *l.CapiLogger) {
		defer waitGroup.Done()

		for !returnDeadStopping {
			returnedToQueue := h.Mb.ReturnDead(int64(h.Env.CapiMqBroker.DeadAfterNoHeartbeatTimeout))
			if len(returnedToQueue) > 0 {
				ReturnDeadTimeoutCounter.Add(float64(len(returnedToQueue)))
				innerLogger.Warn("returned %d stall messages from wip to queue: %s", len(returnedToQueue), strings.Join(returnedToQueue, ";"))
			}
			time.Sleep(1000 * time.Millisecond)
		}
	}(returnedDeadStoppingLogger)

	var prometheusServer *http.Server
	if envConfig.Log.PrometheusExporterPort > 0 {
		logger.Info("listening Prometheus on %d...", envConfig.Log.PrometheusExporterPort)
		prometheus.MustRegister(ApiSuccessHitCounter, ReturnDeadTimeoutCounter)
		prometheusServer = &http.Server{Addr: fmt.Sprintf(":%d", envConfig.Log.PrometheusExporterPort)}
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			http.Handle("/metrics", promhttp.Handler())
			if err := prometheusServer.ListenAndServe(); err != nil {
				log.Fatalf("%s", err.Error())
			}
		}()
	}

	logger.Info("listening API on %d...", h.Env.CapiMqBroker.Port)

	apiServer := &http.Server{Addr: fmt.Sprintf(":%d", h.Env.CapiMqBroker.Port), Handler: h}
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		if err := apiServer.ListenAndServe(); err != nil {
			log.Fatalf("%s", err.Error())
		}
	}()

	// Wait for shutdown signal

	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	for {
		osSignal := <-osSignalChannel
		if osSignal == os.Interrupt || osSignal == os.Kill {
			logger.Info("shutting down...")
			returnDeadStopping = true
			closeApiCtx, closeApiCancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
			if err := apiServer.Shutdown(closeApiCtx); err != nil {
				logger.Error("cannot shutdown API server gracefully: %s", err.Error())
			}
			closeApiCancel()
			if prometheusServer != nil {
				closePrometheusCtx, closePrometheusCancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
				if err := prometheusServer.Shutdown(closePrometheusCtx); err != nil {
					logger.Error("cannot shutdown Prometheus server gracefully: %s", err.Error())
				}
				closePrometheusCancel()
			}
			logger.Info("shutdown complete")
			os.Exit(0)
		}
	}
}

// curl -s -w "\n" -d '[{"script_url":"script1","script_params_url":"scriptparams1","ks":"ks1","run_id":1,"target_node":"node1","first_token":123,"last_token":456,"batch_idx":1,"batches_total":10},{"script_url":"script1","script_params_url":"scriptparams1","ks":"ks1","run_id":1,"target_node":"node1","first_token":457,"last_token":567,"batch_idx":2,"batches_total":10}]' -H "Content-Type: application/json" -X POST http://localhost:7654/q/bulk/
// curl -s 'http://localhost:7654/q/count?waitRetryGroupPrefix=ks1%2F1%2Fnode1'
