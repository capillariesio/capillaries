package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/api"
	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfdb"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	amqp "github.com/rabbitmq/amqp091-go"
)

const port = 6543 // Python Pyramid rules

type route struct {
	method  string
	regex   *regexp.Regexp
	handler http.HandlerFunc
}

func newRoute(method, pattern string, handler http.HandlerFunc) route {
	return route{method, regexp.MustCompile("^" + pattern + "$"), handler}
}

type ApiResponseError struct {
	Msg string `json:"msg"`
}

type ApiResponse struct {
	Data  interface{}      `json:"data"`
	Error ApiResponseError `json:"error"`
}

func WriteApiError(l *l.Logger, w http.ResponseWriter, urlPath string, err error, httpStatus int) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
	l.Error("cannot process %s: %s", urlPath, err.Error())
	respJson, err := json.Marshal(ApiResponse{Error: ApiResponseError{Msg: err.Error()}})
	if err != nil {
		http.Error(w, fmt.Sprintf("unexpected: cannot serialize error response %s", err.Error()), httpStatus)
	} else {
		http.Error(w, string(respJson), httpStatus)
	}
}

func WriteApiSuccess(l *l.Logger, w http.ResponseWriter, data interface{}) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
	respJson, err := json.Marshal(ApiResponse{Data: data})
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot serialize success response: %s", err.Error()), http.StatusInternalServerError)
	} else {
		if _, err := w.Write([]byte(respJson)); err != nil {
			l.Error("cannot write success response, error %s, response %s", err.Error(), respJson)
		}
	}
}

func (h *UrlHandler) ks(w http.ResponseWriter, r *http.Request) {
	cqlSession, err := cql.NewSession(h.Env, "", cql.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	// This works only for Cassandra 4.X, not guaranteed to work for later versions
	qb := cql.QueryBuilder{}
	q := qb.Keyspace("system_schema").Select("keyspaces", []string{"keyspace_name"})
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	respData := make([]string, len(rows))
	ksCount := 0

	for _, row := range rows {
		ks := row["keyspace_name"].(string)
		if len(ks) == 0 || api.CheckKeyspaceName(ks) != nil {
			continue
		}
		respData[ksCount] = ks
		ksCount++
	}

	WriteApiSuccess(h.L, w, respData[:ksCount])
}

type FullRunInfo struct {
	Props   *wfmodel.RunProperties     `json:"props"`
	History []*wfmodel.RunHistoryEvent `json:"history"`
}

// func (h *UrlHandler) ksRuns(w http.ResponseWriter, r *http.Request) {
// 	keyspace := getField(r, 0)
// 	cqlSession, err := cql.NewSession(h.Env, keyspace)
// 	if err != nil {
// 		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}
// 	defer cqlSession.Close()

// 	// Get all runs that were ever registered (maybe even not started)
// 	allRunsProps, err := wfdb.GetAllRunsProperties(cqlSession, keyspace)
// 	if err != nil {
// 		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}

// 	// Make a ref map RunId->FullRunInfo
// 	resultMap := map[int16]*FullRunInfo{}
// 	for _, runProps := range allRunsProps {
// 		resultMap[runProps.RunId] = &FullRunInfo{Props: runProps}
// 	}

// 	// Get run history
// 	runs, err := api.GetRunHistory(h.L, cqlSession, keyspace)
// 	if err != nil {
// 		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}

// 	// Enrich result with history
// 	for _, run := range runs {
// 		fullRunInfo := resultMap[run.RunId]
// 		fullRunInfo.History = append(fullRunInfo.History, run)
// 	}

// 	// Map to list
// 	result := make([]*FullRunInfo, len(resultMap))
// 	for idx, runProps := range allRunsProps {
// 		result[idx] = resultMap[runProps.RunId]
// 	}

// 	WriteApiSuccess(h.L, w, result)
// }

// func (h *UrlHandler) ksNodes(w http.ResponseWriter, r *http.Request) {
// 	keyspace := getField(r, 0)
// 	cqlSession, err := cql.NewSession(h.Env, keyspace)
// 	if err != nil {
// 		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}
// 	defer cqlSession.Close()

// 	result, err := api.GetRunsNodeHistory(h.L, cqlSession, keyspace, []int16{})
// 	if err != nil {
// 		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}
// 	WriteApiSuccess(h.L, w, result)
// }

// type WebapiRunStatus struct {
// 	RunId  int16                 `json:"run_id"`
// 	Status wfmodel.RunStatusType `json:"status"`
// 	Ts     string                `json:"ts"`
// }

type WebapiNodeStatus struct {
	RunId  int16                       `json:"run_id"`
	Status wfmodel.NodeBatchStatusType `json:"status"`
	Ts     string                      `json:"ts"`
}

type WebapiNodeRunMatrixRow struct {
	NodeName     string             `json:"node_name"`
	NodeStatuses []WebapiNodeStatus `json:"node_statuses"`
}
type WebapiNodeRunMatrix struct {
	RunLifespans []*wfmodel.RunLifespan   `json:"run_lifespans"`
	Nodes        []WebapiNodeRunMatrixRow `json:"nodes"`
}

func (h *UrlHandler) ksMatrix(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace, cql.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	// Retrieve all runs that happened in this ks and find their current statuses
	runLifespanMap, err := wfdb.HarvestRunLifespans(h.L, cqlSession, keyspace, []int16{})
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// Arrange run statuses for the matrix header
	mx := WebapiNodeRunMatrix{RunLifespans: make([]*wfmodel.RunLifespan, len(runLifespanMap))}
	runCount := 0
	for _, runLifespan := range runLifespanMap {
		mx.RunLifespans[runCount] = runLifespan
		runCount++
	}
	sort.Slice(mx.RunLifespans, func(i, j int) bool { return mx.RunLifespans[i].RunId < mx.RunLifespans[j].RunId })

	// Retireve all node events for this ks, for all runs
	nodeHistory, err := api.GetRunsNodeHistory(h.L, cqlSession, keyspace, []int16{})
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// For each node/run, harvest current node status, latest wins
	nodeRunStatusMap := map[string]map[int16]WebapiNodeStatus{}
	nodeStartTsMap := map[string]time.Time{}
	for _, nodeEvent := range nodeHistory {
		if _, ok := nodeRunStatusMap[nodeEvent.ScriptNode]; !ok {
			nodeRunStatusMap[nodeEvent.ScriptNode] = map[int16]WebapiNodeStatus{}
		}
		nodeRunStatusMap[nodeEvent.ScriptNode][nodeEvent.RunId] = WebapiNodeStatus{RunId: nodeEvent.RunId, Status: nodeEvent.Status, Ts: nodeEvent.Ts.Format("2006-01-02T15:04:05.000-0700")}

		if _, ok := nodeStartTsMap[nodeEvent.ScriptNode]; !ok {
			nodeStartTsMap[nodeEvent.ScriptNode] = nodeEvent.Ts
		}
	}

	// Arrange status in the result mx
	mx.Nodes = make([]WebapiNodeRunMatrixRow, len(nodeRunStatusMap))
	nodeCount := 0
	for nodeName, runNodeStatusMap := range nodeRunStatusMap {
		mx.Nodes[nodeCount] = WebapiNodeRunMatrixRow{NodeName: nodeName, NodeStatuses: make([]WebapiNodeStatus, len(mx.RunLifespans))}
		for runIdx, matrixRunLifespan := range mx.RunLifespans {
			if nodeStatus, ok := runNodeStatusMap[matrixRunLifespan.RunId]; ok {
				mx.Nodes[nodeCount].NodeStatuses[runIdx] = nodeStatus
			}
		}
		nodeCount++
	}

	// Sort nodes: those who were processed (including started) earlier, go first
	sort.Slice(mx.Nodes, func(i, j int) bool {
		return nodeStartTsMap[mx.Nodes[i].NodeName].Before(nodeStartTsMap[mx.Nodes[j].NodeName])
	})

	WriteApiSuccess(h.L, w, mx)
}

// func (h *UrlHandler) ksRunNodes(w http.ResponseWriter, r *http.Request) {
// 	keyspace := getField(r, 0)
// 	cqlSession, err := cql.NewSession(h.Env, keyspace)
// 	if err != nil {
// 		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}
// 	defer cqlSession.Close()

// 	runId, err := strconv.Atoi(getField(r, 1))
// 	if err != nil {
// 		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}

// 	result, err := api.GetRunsNodeHistory(h.L, cqlSession, keyspace, []int16{int16(runId)})
// 	if err != nil {
// 		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}
// 	WriteApiSuccess(h.L, w, result)
// }

type RunNodeBatchesInfo struct {
	RunProps            *wfmodel.RunProperties       `json:"run_props"`
	RunLs               *wfmodel.RunLifespan         `json:"run_lifespan"`
	RunNodeBatchHistory []*wfmodel.BatchHistoryEvent `json:"batch_history"`
}

func (h *UrlHandler) ksRunNodeBatchHistory(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace, cql.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	runId, err := strconv.Atoi(getField(r, 1))
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// Static run properties
	// TODO: consider caching

	allRunsProps, err := wfdb.GetRunProperties(cqlSession, keyspace, int16(runId))
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	if len(allRunsProps) != 1 {
		WriteApiError(h.L, w, r.URL.Path, fmt.Errorf("invalid number of matching runs (%d), expected 1 ", len(allRunsProps)), http.StatusInternalServerError)
		return
	}

	result := RunNodeBatchesInfo{RunProps: allRunsProps[0]}

	// Run status

	runLifeSpans, err := wfdb.HarvestRunLifespans(h.L, cqlSession, keyspace, []int16{int16(runId)})
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	if len(runLifeSpans) != 1 {
		WriteApiError(h.L, w, r.URL.Path, fmt.Errorf("invalid number of run life spans (%d), expected 1 ", len(runLifeSpans)), http.StatusInternalServerError)
		return
	}
	result.RunLs = runLifeSpans[int16(runId)]

	// Batch history

	nodeName := getField(r, 2)
	result.RunNodeBatchHistory, err = wfdb.GetRunNodeBatchHistory(h.L, cqlSession, keyspace, int16(runId), nodeName)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	WriteApiSuccess(h.L, w, result)
}

type RunNodesInfo struct {
	RunProps *wfmodel.RunProperties `json:"run_props"`
	RunLs    *wfmodel.RunLifespan   `json:"run_lifespan"`
}

func (h *UrlHandler) ksRunNodeHistory(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace, cql.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	runId, err := strconv.Atoi(getField(r, 1))
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// Static run properties
	// TODO: consider caching

	allRunsProps, err := wfdb.GetRunProperties(cqlSession, keyspace, int16(runId))
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	if len(allRunsProps) != 1 {
		WriteApiError(h.L, w, r.URL.Path, fmt.Errorf("invalid number of matching runs (%d), expected 1 ", len(allRunsProps)), http.StatusInternalServerError)
		return
	}

	result := RunNodesInfo{RunProps: allRunsProps[0]}

	// Run status

	runLifeSpans, err := wfdb.HarvestRunLifespans(h.L, cqlSession, keyspace, []int16{int16(runId)})
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	if len(runLifeSpans) != 1 {
		WriteApiError(h.L, w, r.URL.Path, fmt.Errorf("invalid number of run life spans (%d), expected 1 ", len(runLifeSpans)), http.StatusInternalServerError)
		return
	}
	result.RunLs = runLifeSpans[int16(runId)]

	// TODO: node history

	WriteApiSuccess(h.L, w, result)
}

type StartRunInfo struct {
	RunId int16 `json:"run_id"`
}

func (h *UrlHandler) ksStartRun(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace, cql.CreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	amqpConnection, err := amqp.Dial(h.Env.Amqp.URL)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, fmt.Errorf("cannot dial RabbitMQ at %v, will reconnect: %v\n", h.Env.Amqp.URL, err), http.StatusInternalServerError)
		return
	}
	defer amqpConnection.Close()

	amqpChannel, err := amqpConnection.Channel()
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, fmt.Errorf("cannot create amqp channel: %v\n", err), http.StatusInternalServerError)
		return
	}
	defer amqpChannel.Close()

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runProps := wfmodel.RunProperties{}
	if err = json.Unmarshal(bodyBytes, &runProps); err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runId, err := api.StartRun(h.Env, h.L, amqpChannel, runProps.ScriptUri, runProps.ScriptParamsUri, cqlSession, keyspace, strings.Split(runProps.StartNodes, ","))
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	WriteApiSuccess(h.L, w, StartRunInfo{RunId: runId})
}

type StopRunInfo struct {
	Comment string `json:"comment"`
}

func (h *UrlHandler) ksStopRunOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,DELETE")
	WriteApiSuccess(h.L, w, nil)
}

func (h *UrlHandler) ksStopRun(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace, cql.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	stopRunInfo := StopRunInfo{}
	if err = json.Unmarshal(bodyBytes, &stopRunInfo); err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runId, err := strconv.Atoi(getField(r, 1))
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	if err = api.StopRun(h.L, cqlSession, keyspace, int16(runId), stopRunInfo.Comment); err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	WriteApiSuccess(h.L, w, nil)
}

type UrlHandler struct {
	Env *env.EnvConfig
	L   *l.Logger
}

type ctxKey struct {
}

func getField(r *http.Request, index int) string {
	fields := r.Context().Value(ctxKey{}).([]string)
	return fields[index]
}

var routes []route

func (h UrlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func main() {
	envConfig, err := env.ReadEnvConfigFile("env_config.json")
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}

	logger, err := l.NewLoggerFromEnvConfig(envConfig)
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}
	defer logger.Close()

	mux := http.NewServeMux()

	h := UrlHandler{Env: envConfig, L: logger}

	routes = []route{
		newRoute("GET", "/ks[/]*", h.ks),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)[/]*", h.ksMatrix),
		//newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run[/]*", h.ksRuns),
		//newRoute("GET", "/ks/([a-zA-Z0-9_]+)/node[/]*", h.ksNodes),
		//newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)/node[/]*", h.ksRunNodes),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)/node/([a-zA-Z0-9_]+)/batch_history[/]*", h.ksRunNodeBatchHistory),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)/node_history[/]*", h.ksRunNodeHistory),
		newRoute("POST", "/ks/([a-zA-Z0-9_]+)/run[/]*", h.ksStartRun),
		newRoute("DELETE", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)[/]*", h.ksStopRun),
		newRoute("OPTIONS", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)[/]*", h.ksStopRunOptions),
	}

	mux.Handle("/", h)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}
}
