package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/mq_message_broker"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type UrlHandlerInstance struct {
	Env *env.EnvConfig
	L   *l.CapiLogger
	Mb  *mq_message_broker.MessageBroker
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

type ApiResponseError struct {
	Msg string `json:"msg"`
}

type ApiResponse struct {
	Data  any              `json:"data"`
	Error ApiResponseError `json:"error"`
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

	w.Header().Set("Access-Control-Allow-Origin", pickAccessControlAllowOrigin(wc, r))
	logger.Error("cannot process %s: %s", urlPath, err.Error())
	respJson, err := json.Marshal(ApiResponse{Error: ApiResponseError{Msg: err.Error()}})
	if err != nil {
		http.Error(w, fmt.Sprintf("unexpected: cannot serialize error response %s", err.Error()), httpStatus)
	} else {
		http.Error(w, string(respJson), httpStatus)
	}
}

func WriteApiSuccess(logger *l.CapiLogger, wc *env.CapiMqBrokerConfig, r *http.Request, w http.ResponseWriter, data any) {
	logger.PushF("WriteApiSuccess")
	defer logger.PopF()

	logger.Debug("%s: OK", r.URL.Path)

	w.Header().Set("Access-Control-Allow-Origin", pickAccessControlAllowOrigin(wc, r))
	respJson, err := json.Marshal(ApiResponse{Data: data})
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

	var msgs []*wfmodel.Message
	if err = json.Unmarshal(bodyBytes, &msgs); err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	if err := h.Mb.QBulk(msgs, h.Env.CapiMqBroker.MaxMessages); err != nil {
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

	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, msg)
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

func (h *UrlHandlerInstance) ks(w http.ResponseWriter, r *http.Request) {
	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, h.Mb.Ks())
}

func getHeapTypeFromRequest(r *http.Request) (mq_message_broker.HeapType, error) {
	qOrWip, err := getFieldByIndexFromRequest(r, 0)
	if err != nil {
		return mq_message_broker.HeapTypeUnknown, err
	}

	return mq_message_broker.StringToHeapType(qOrWip)
}

func getQueueStartFromRequest(r *http.Request) (mq_message_broker.QueueReadType, error) {
	headOrTail, err := getFieldByIndexFromRequest(r, 1)
	if err != nil {
		return mq_message_broker.QueueReadUnknown, err
	}

	return mq_message_broker.StringToQueueReadType(headOrTail)
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

func getMsgParamsFromQuery(r *http.Request) (string, int16, string, error) {
	ks := r.URL.Query().Get("ks")
	run_id_string := r.URL.Query().Get("run_id")
	run_id := int64(0)
	var err error
	if len(run_id_string) > 0 {
		run_id, err = strconv.ParseInt(run_id_string, 10, 64)
		if err != nil {
			return "", 0, "", fmt.Errorf("invalid run_id parameter %s", run_id_string)
		}
	}
	target_node := r.URL.Query().Get("target_node")
	return ks, int16(run_id), target_node, nil
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

	ks, runId, nodeName, err := getMsgParamsFromQuery(r)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, h.Mb.Count(heapType, ks, runId, nodeName))
}

func (h *UrlHandlerInstance) delete(w http.ResponseWriter, r *http.Request) {
	heapType, err := getHeapTypeFromRequest(r)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	ks, runId, nodeName, err := getMsgParamsFromQuery(r)
	if err != nil {
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, h.Mb.Delete(heapType, ks, runId, nodeName))
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
	case mq_message_broker.QueueReadHead, mq_message_broker.QueueReadTail:
		from, count, err := getFromCountParamsFromQuery(r)
		if err != nil {
			WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		}
		WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, h.Mb.HeadTail(heapType, queueRead, from, count))
	case mq_message_broker.QueueReadFilter:
		ks, runId, nodeName, err := getMsgParamsFromQuery(r)
		if err != nil {
			WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, err, http.StatusInternalServerError)
		}
		WriteApiSuccess(h.L, &h.Env.CapiMqBroker, r, w, h.Mb.Filter(heapType, ks, runId, nodeName))
	default:
		WriteApiError(h.L, &h.Env.CapiMqBroker, r, w, r.URL.Path, fmt.Errorf("unexpected queueRead %s", queueRead), http.StatusInternalServerError)
	}
}

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

	h := UrlHandlerInstance{Env: envConfig, L: logger, Mb: mq_message_broker.NewMessageBroker()}

	routes = []route{
		newRoute("POST", "/q/bulk[/]*", h.qBulk),
		newRoute("POST", "/q/claim[/]*", h.qClaim),
		newRoute("DELETE", "/wip/ack/([A-Fa-f0-9-]+)[/]*", h.wipAck),
		newRoute("POST", "/wip/hearbeat/([A-Fa-f0-9-]+)[/]*", h.wipHeartbeat),
		newRoute("POST", "/wip/return/([A-Fa-f0-9-]+)[/]*", h.wipReturn),

		newRoute("GET", "/ks[/]*", h.ks),
		newRoute("GET", "/(q|wip)/count[/]*", h.count),
		newRoute("DELETE", "/(q|wip)[/]*", h.delete),
		newRoute("GET", "/(q|wip)/(head|tail|filter)[/]*", h.headTailFilter),
	}

	mux.Handle("/", h)

	if envConfig.Log.PrometheusExporterPort > 0 {
		prometheus.MustRegister(xfer.SftpFileGetGetDuration, xfer.HttpFileGetGetDuration, xfer.S3FileGetGetDuration)
		prometheus.MustRegister(sc.ScriptDefCacheHitCounter, sc.ScriptDefCacheMissCounter)
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(fmt.Sprintf(":%d", envConfig.Log.PrometheusExporterPort), nil); err != nil {
				log.Fatalf("%s", err.Error())
			}
		}()
	}

	logger.Info("listening on %d...", h.Env.CapiMqBroker.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", h.Env.CapiMqBroker.Port), mux); err != nil {
		log.Fatalf("%s", err.Error())
	}
}

// curl -s -w "\n" -d '[{"script_url":"script1","script_params_url":"scriptparams1","ks":"ks1","run_id":1,"target_node":"node1","first_token":123,"last_token":456,"batch_idx":1,"batches_total":10},{"script_url":"script1","script_params_url":"scriptparams1","ks":"ks1","run_id":1,"target_node":"node1","first_token":457,"last_token":567,"batch_idx":2,"batches_total":10}]' -H "Content-Type: application/json" -X POST http://localhost:7654/q/bulk/
// curl -s 'http://localhost:7654/q/count?ks=ks1&run_id=1&target_node=node1'
