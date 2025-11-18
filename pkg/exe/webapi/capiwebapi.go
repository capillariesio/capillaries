package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/capillariesio/capillaries/pkg/api"
	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/custom/py_calc"
	"github.com/capillariesio/capillaries/pkg/custom/tag_and_denormalize"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/mq"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type StandardWebapiProcessorDefFactory struct {
}

func (f *StandardWebapiProcessorDefFactory) Create(processorType string) (sc.CustomProcessorDef, bool) {
	// All processors to be supported by this 'stock' binary (daemon/toolbelt/webapi).
	// If you develop your own processor(s), use your own ProcessorDefFactory that lists all processors,
	// they all must implement CustomProcessorRunner interface
	switch processorType {
	case py_calc.ProcessorPyCalcName:
		return &py_calc.PyCalcProcessorDef{}, true
	case tag_and_denormalize.ProcessorTagAndDenormalizeName:
		return &tag_and_denormalize.TagAndDenormalizeProcessorDef{}, true
	default:
		return nil, false
	}
}

type route struct {
	method  string
	regex   *regexp.Regexp
	handler http.HandlerFunc
}

func newRoute(method, pattern string, handler http.HandlerFunc) route {
	return route{method, regexp.MustCompile("^" + pattern + "$"), handler}
}

type ApiResponse struct {
	Data  any    `json:"data"`
	Error string `json:"error"` // If need more, consider re-introducing ApiResponseError struct, and watch client side responseJson
}

func pickAccessControlAllowOrigin(wc *env.WebapiConfig, r *http.Request) string {
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

func WriteApiError(logger *l.CapiLogger, wc *env.WebapiConfig, r *http.Request, w http.ResponseWriter, urlPath string, err error, httpStatus int) {
	logger.PushF("WriteApiError")
	defer logger.PopF()

	w.Header().Set("Access-Control-Allow-Origin", pickAccessControlAllowOrigin(wc, r))
	logger.Error("cannot process %s: %s", urlPath, err.Error())
	respJson, err := json.Marshal(ApiResponse{Error: err.Error()})
	if err != nil {
		http.Error(w, fmt.Sprintf("unexpected: cannot serialize error response %s", err.Error()), httpStatus)
	} else {
		http.Error(w, string(respJson), httpStatus)
	}
}

func WriteApiSuccess(logger *l.CapiLogger, wc *env.WebapiConfig, r *http.Request, w http.ResponseWriter, data any) {
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

func (h *UrlHandler) ks(w http.ResponseWriter, r *http.Request) {
	cqlSession, _, err := db.NewSession(h.Env, "", db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	// This works only for Cassandra 4.X, not guaranteed to work for later versions
	qb := cql.QueryBuilder{}
	q := qb.Keyspace("system_schema").Select("keyspaces", []string{"keyspace_name"})
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	respData := make([]string, len(rows))
	ksCount := 0

	for _, row := range rows {
		ksVolatile, ok := row["keyspace_name"]
		if !ok {
			WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, errors.New("cannot find keyspace_name in the response"), http.StatusInternalServerError)
			return
		}

		ks, ok := ksVolatile.(string)
		if !ok {
			WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, fmt.Errorf("cannot cast keyspace_name to string: %v", row["keyspace_name"]), http.StatusInternalServerError)
			return
		}
		if len(ks) == 0 || api.IsSystemKeyspaceName(ks) {
			continue
		}
		respData[ksCount] = ks
		ksCount++
	}

	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, respData[:ksCount])
}

type FullRunInfo struct {
	Props   *wfmodel.RunProperties     `json:"props"`
	History []*wfmodel.RunHistoryEvent `json:"history"`
}

type WebapiNodeStatus struct {
	RunId  int16                       `json:"run_id"`
	Status wfmodel.NodeBatchStatusType `json:"status"`
	Ts     string                      `json:"ts"`
}

type WebapiNodeRunMatrixRow struct {
	NodeName     string             `json:"node_name"`
	NodeDesc     string             `json:"node_desc"`
	NodeStatuses []WebapiNodeStatus `json:"node_statuses"`
}
type WebapiNodeRunMatrix struct {
	RunLifespans []*wfmodel.RunLifespan   `json:"run_lifespans"`
	Nodes        []WebapiNodeRunMatrixRow `json:"nodes"`
}

// Poor man's cache
var NodeDescCache = map[string]string{}
var NodeDescCacheTs = map[string]time.Time{}
var NodeDescCacheLock = sync.RWMutex{}

func (h *UrlHandler) getNodeDesc(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runId int16, nodeName string) (string, error) {

	nodeKey := keyspace + ":" + nodeName
	NodeDescCacheLock.RLock()
	nodeDesc, okData := NodeDescCache[nodeKey]
	nodeDescTs, okTs := NodeDescCacheTs[nodeKey]
	NodeDescCacheLock.RUnlock()
	if okData && okTs && time.Since(nodeDescTs).Seconds() < 30 {
		return nodeDesc, nil
	}

	// Static run props

	runProps, err := getRunProps(logger, cqlSession, keyspace, runId)
	if err != nil {
		return "", err
	}

	// Now we have script URL, load it

	script, _, err := sc.NewScriptFromFiles(h.Env.CaPath, h.Env.PrivateKeys, runProps.ScriptUrl, runProps.ScriptParamsUrl, h.Env.CustomProcessorDefFactoryInstance, h.Env.CustomProcessorsSettings)
	if err != nil {
		return "", err
	}

	nodeDef, ok := script.ScriptNodes[nodeName]
	if !ok {
		return "", fmt.Errorf("cannot find node %s", nodeName)
	}

	NodeDescCacheLock.RLock()
	if len(NodeDescCache) > 1000 {
		for k := range NodeDescCache {
			delete(NodeDescCache, k)
		}
	}
	NodeDescCache[nodeKey] = nodeDef.Desc
	NodeDescCacheTs[nodeKey] = time.Now()
	NodeDescCacheLock.RUnlock()

	return nodeDef.Desc, nil
}

func (h *UrlHandler) ksMatrix(w http.ResponseWriter, r *http.Request) {
	keyspace, err := getField(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	cqlSession, _, err := db.NewSession(h.Env, keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	// Retrieve all runs that happened in this ks and find their current statuses
	runLifespanMap, err := api.HarvestRunLifespans(h.L, cqlSession, keyspace, []int16{})
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
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

	// Retrieve all node events for this ks, for all runs
	nodeHistory, err := api.GetNodeHistoryForRuns(h.L, cqlSession, keyspace, []int16{})
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	nodeStartTsMap := map[string]time.Time{} // Arrange by the ts in the last run

	// For each node/run, harvest current node status, latest wins
	nodeRunStatusMap := map[string]map[int16]WebapiNodeStatus{}
	for _, nodeEvent := range nodeHistory {
		if _, ok := nodeRunStatusMap[nodeEvent.ScriptNode]; !ok {
			nodeRunStatusMap[nodeEvent.ScriptNode] = map[int16]WebapiNodeStatus{}
		}
		nodeRunStatusMap[nodeEvent.ScriptNode][nodeEvent.RunId] = WebapiNodeStatus{RunId: nodeEvent.RunId, Status: nodeEvent.Status, Ts: nodeEvent.Ts.Format("2006-01-02T15:04:05.000-0700")}

		if nodeEvent.Status == wfmodel.NodeBatchStart {
			if _, ok := nodeStartTsMap[nodeEvent.ScriptNode]; !ok {
				nodeStartTsMap[nodeEvent.ScriptNode] = nodeEvent.Ts
			}
		}
	}

	// Arrange status in the result mx
	mx.Nodes = make([]WebapiNodeRunMatrixRow, len(nodeRunStatusMap))
	nodeCount := 0
	for nodeName, runNodeStatusMap := range nodeRunStatusMap {
		nodeDesc, err := h.getNodeDesc(h.L, cqlSession, keyspace, runLifespanMap[1].RunId, nodeName)
		if err != nil {
			WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, fmt.Errorf("cannot get node description: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		mx.Nodes[nodeCount] = WebapiNodeRunMatrixRow{NodeName: nodeName, NodeDesc: nodeDesc, NodeStatuses: make([]WebapiNodeStatus, len(mx.RunLifespans))}
		for runIdx, matrixRunLifespan := range mx.RunLifespans {
			if nodeStatus, ok := runNodeStatusMap[matrixRunLifespan.RunId]; ok {
				mx.Nodes[nodeCount].NodeStatuses[runIdx] = nodeStatus
			}
		}
		nodeCount++
	}

	// Sort nodes: started come first, sorted by start ts, other come after that, sorted by node name
	// Ideally, they should be sorted geometrically from DAG, with start ts coming into play when DAG says nodes are equal.
	// But this will require script analysis which takes too long.
	sort.Slice(mx.Nodes, func(i, j int) bool {
		leftTs, leftPresent := nodeStartTsMap[mx.Nodes[i].NodeName]
		rightTs, rightPresent := nodeStartTsMap[mx.Nodes[j].NodeName]
		if !leftPresent && rightPresent {
			return false
		} else if leftPresent && !rightPresent {
			return true
		} else if leftPresent && rightPresent && !leftTs.Equal(rightTs) {
			return leftTs.Before(rightTs)
		}

		// Sort by node name
		return mx.Nodes[i].NodeName < mx.Nodes[j].NodeName
	})

	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, mx)
}

// Poor man's cache
var RunPropsCache = map[string]*wfmodel.RunProperties{}
var RunPropsCacheTs = map[string]time.Time{}
var RunPropsCacheLock = sync.RWMutex{}

func getRunProps(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runId int16) (*wfmodel.RunProperties, error) {
	runPropsCacheKey := keyspace + ":" + fmt.Sprintf("%d", runId)
	RunPropsCacheLock.RLock()
	oneRunProps, okData := RunPropsCache[runPropsCacheKey]
	oneRunPropsTs, okTs := RunPropsCacheTs[runPropsCacheKey]
	RunPropsCacheLock.RUnlock()

	if okData && okTs && time.Since(oneRunPropsTs).Seconds() < 30 {
		return oneRunProps, nil
	}
	allRunsProps, err := api.GetRunProperties(logger, cqlSession, keyspace, int16(runId))
	if err != nil {
		return nil, err
	}
	if len(allRunsProps) != 1 {
		return nil, fmt.Errorf("invalid number of matching runs (%d), expected 1; this usually happens when webapi caller makes wrong assumptions about the process status", len(allRunsProps))
	}

	RunPropsCacheLock.Lock()
	if len(RunPropsCache) > 1000 {
		for k := range RunPropsCache {
			delete(RunPropsCache, k)
		}
	}
	RunPropsCache[runPropsCacheKey] = allRunsProps[0]
	RunPropsCacheTs[runPropsCacheKey] = time.Now()
	RunPropsCacheLock.Unlock()

	return allRunsProps[0], nil
}

func getRunPropsAndLifespans(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runId int16) (*wfmodel.RunProperties, *wfmodel.RunLifespan, error) {

	// Static run props

	runProps, err := getRunProps(logger, cqlSession, keyspace, runId)
	if err != nil {
		return nil, nil, err
	}

	// Run status

	runLifeSpans, err := api.HarvestRunLifespans(logger, cqlSession, keyspace, []int16{int16(runId)})
	if err != nil {
		return nil, nil, err
	}
	if len(runLifeSpans) != 1 {
		return nil, nil, fmt.Errorf("invalid number of run life spans (%d), expected 1 ", len(runLifeSpans))
	}

	return runProps, runLifeSpans[int16(runId)], nil
}

type RunNodeBatchesInfo struct {
	RunProps            *wfmodel.RunProperties       `json:"run_props"`
	RunLs               *wfmodel.RunLifespan         `json:"run_lifespan"`
	RunNodeBatchHistory []*wfmodel.BatchHistoryEvent `json:"batch_history"`
}

func (h *UrlHandler) ksRunNodeBatchHistory(w http.ResponseWriter, r *http.Request) {
	keyspace, err := getField(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	cqlSession, _, err := db.NewSession(h.Env, keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	runIdString, err := getField(r, 1)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runId, err := strconv.Atoi(runIdString)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	result := RunNodeBatchesInfo{}
	result.RunProps, result.RunLs, err = getRunPropsAndLifespans(h.L, cqlSession, keyspace, int16(runId))
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// Batch history

	nodeName, err := getField(r, 2)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	result.RunNodeBatchHistory, err = api.GetRunNodeBatchHistory(h.L, cqlSession, keyspace, int16(runId), nodeName)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, result)
}

type RunNodesInfo struct {
	RunProps       *wfmodel.RunProperties      `json:"run_props"`
	RunLs          *wfmodel.RunLifespan        `json:"run_lifespan"`
	RunNodeHistory []*wfmodel.NodeHistoryEvent `json:"node_history"`
}

func (h *UrlHandler) ksRunNodeHistory(w http.ResponseWriter, r *http.Request) {
	keyspace, err := getField(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	cqlSession, _, err := db.NewSession(h.Env, keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	runIdString, err := getField(r, 1)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runId, err := strconv.Atoi(runIdString)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	result := RunNodesInfo{}
	result.RunProps, result.RunLs, err = getRunPropsAndLifespans(h.L, cqlSession, keyspace, int16(runId))
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// Node history

	result.RunNodeHistory, err = api.GetNodeHistoryForRun(h.L, cqlSession, keyspace, int16(runId))
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	sort.Slice(result.RunNodeHistory, func(i, j int) bool { return result.RunNodeHistory[i].Ts.Before(result.RunNodeHistory[j].Ts) })

	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, result)
}

func (h *UrlHandler) ksRunViz(w http.ResponseWriter, r *http.Request) {
	var useRootPalette bool
	useRootPaletteParam := r.URL.Query().Get("use_root_palette")
	if useRootPaletteParam == "true" {
		useRootPalette = true
	}

	var isStatus bool
	isStatusParam := r.URL.Query().Get("is_status")
	if isStatusParam == "true" {
		isStatus = true
	}

	keyspace, err := getField(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	cqlSession, _, err := db.NewSession(h.Env, keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	runIdString, err := getField(r, 1)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runId, err := strconv.Atoi(runIdString)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// Extract script URL from run props
	runProps, err := getRunProps(h.L, cqlSession, keyspace, int16(runId))
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	// Now we have script URL, load it
	scriptDef, _, err := sc.NewScriptFromFiles(h.Env.CaPath, h.Env.PrivateKeys, runProps.ScriptUrl, runProps.ScriptParamsUrl, h.Env.CustomProcessorDefFactoryInstance, h.Env.CustomProcessorsSettings)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
	}

	var nodeColorMap map[string]int32
	showIdx := true
	showFields := true
	if isStatus {
		nodeColorMap = map[string]int32{}
		nodes, err := api.GetNodeHistoryForRuns(h.L, cqlSession, keyspace, []int16{int16(runId)})
		if err != nil {
			WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		}
		for _, node := range nodes {
			nodeColorMap[node.ScriptNode] = api.NodeBatchStatusToCapigraphColor(node.Status)
		}
		showIdx = false
		showFields = false
		useRootPalette = false
	}

	svg := api.GetCapigraphDiagram(scriptDef, showIdx, showFields, useRootPalette, nodeColorMap)

	w.Header().Set("content-type", "image/svg+xml")
	w.Header().Set("Access-Control-Allow-Origin", pickAccessControlAllowOrigin(&h.Env.Webapi, r))
	if _, err := w.Write([]byte(svg)); err != nil {
		h.L.Error("cannot write svg response, error %s", err.Error())
	}
}

// func (h *UrlHandler) ksRunStatusViz(w http.ResponseWriter, r *http.Request) {
// 	keyspace, err := getField(r, 0)
// 	if err != nil {
// 		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}

// 	cqlSession, err := db.NewSession(h.Env, keyspace, db.DoNotCreateKeyspaceOnConnect)
// 	if err != nil {
// 		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}
// 	defer cqlSession.Close()

// 	runIdString, err := getField(r, 1)
// 	if err != nil {
// 		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}

// 	runId, err := strconv.Atoi(runIdString)
// 	if err != nil {
// 		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}

// 	svg, err := h.getViz(h.ScriptCache, h.L, cqlSession, keyspace, int16(runId), true)
// 	if err != nil {
// 		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("content-type", "image/svg+xml")
// 	w.Header().Set("Access-Control-Allow-Origin", pickAccessControlAllowOrigin(&h.Env.Webapi, r))
// 	if _, err := w.Write([]byte(svg)); err != nil {
// 		h.L.Error("cannot write svg response, error %s", err.Error())
// 	}
// }

type StartedRunInfo struct {
	RunId int16 `json:"run_id"`
}

func (h *UrlHandler) ksStartRunOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,POST")
	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, nil)
}

func (h *UrlHandler) ksStartRun(w http.ResponseWriter, r *http.Request) {
	keyspace, err := getField(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	dbStartTime := time.Now()
	cqlSession, cassandraEngine, err := db.NewSession(h.Env, keyspace, db.CreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	h.L.Info("start run in %s, db session creation took %.2fs", keyspace, time.Since(dbStartTime).Seconds())

	amqpStartTime := time.Now()

	var mqProducer mq.MqProducer
	if h.Env.MqType == string(mq.MqClientCapimq) {
		mqProducer = mq.NewCapimqProducer(h.Env.CapiMqClient.URL)
	} else {
		mqProducer = mq.NewAmqp10Producer(h.Env.Amqp10.URL, h.Env.Amqp10.Address)
	}

	err = mqProducer.Open()
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, fmt.Errorf("cannot open mq: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer mqProducer.Close()

	h.L.Info("start runing %s, mq connect took %.2fs", keyspace, time.Since(amqpStartTime).Seconds())

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runProps := wfmodel.RunProperties{}
	if err = json.Unmarshal(bodyBytes, &runProps); err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	//runId, err := api.StartRun(h.Env, h.L, amqpChannel, mqSender, runProps.ScriptUrl, runProps.ScriptParamsUrl, cqlSession, cassandraEngine, keyspace, strings.Split(runProps.StartNodes, ","), runProps.RunDescription)
	runId, err := api.StartRun(h.Env, h.L, mqProducer, runProps.ScriptUrl, runProps.ScriptParamsUrl, cqlSession, cassandraEngine, keyspace, strings.Split(runProps.StartNodes, ","), runProps.RunDescription)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, StartedRunInfo{RunId: runId})
}

type StopRunInfo struct {
	Comment string `json:"comment"`
}

func (h *UrlHandler) ksStopRunOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,DELETE")
	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, nil)
}

func (h *UrlHandler) ksStopRun(w http.ResponseWriter, r *http.Request) {
	keyspace, err := getField(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	cqlSession, _, err := db.NewSession(h.Env, keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	stopRunInfo := StopRunInfo{}
	if err = json.Unmarshal(bodyBytes, &stopRunInfo); err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runIdString, err := getField(r, 1)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runId, err := strconv.Atoi(runIdString)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	if err = api.StopRun(h.L, cqlSession, keyspace, int16(runId), stopRunInfo.Comment); err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, nil)
}

func (h *UrlHandler) ksDropOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,DELETE")
	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, nil)
}

func (h *UrlHandler) ksDrop(w http.ResponseWriter, r *http.Request) {
	keyspace, err := getField(r, 0)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	cqlSession, _, err := db.NewSession(h.Env, keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	err = api.DropKeyspace(h.L, cqlSession, keyspace)
	if err != nil {
		WriteApiError(h.L, &h.Env.Webapi, r, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	WriteApiSuccess(h.L, &h.Env.Webapi, r, w, nil)
}

type UrlHandler struct {
	Env *env.EnvConfig
	L   *l.CapiLogger
}

type ctxKey struct {
}

func getField(r *http.Request, index int) (string, error) {
	fields, ok := r.Context().Value(ctxKey{}).([]string)
	if !ok {
		return "", errors.New("no fields in http request")
	}
	if len(fields) <= index {
		return "", fmt.Errorf("no t enough fields in http request, index %d", index)
	}
	return fields[index], nil
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

var version string

func main() {
	initCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	envConfig, err := env.ReadEnvConfigFile(initCtx, "capiwebapi.json")
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	// Webapi (like toolbelt and daemon) requires custom proc def factory, otherwise it will not be able to start runs
	envConfig.CustomProcessorDefFactoryInstance = &StandardWebapiProcessorDefFactory{}
	logger, err := l.NewLoggerFromEnvConfig(envConfig)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	defer logger.Close()

	logger.Info("Capillaries webapi %s", version)
	logger.Info("env config: %s", envConfig.String())
	logger.Info("S3 config status: %s", xfer.GetS3ConfigStatus(initCtx).String())

	mux := http.NewServeMux()

	h := UrlHandler{Env: envConfig, L: logger}

	routes = []route{
		newRoute("GET", "/ks[/]*", h.ks),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)[/]*", h.ksMatrix),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)/node/([a-zA-Z0-9_]+)/batch_history[/]*", h.ksRunNodeBatchHistory),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)/node_history[/]*", h.ksRunNodeHistory),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)/viz[/\\?&=a-zA-Z0-9_]*", h.ksRunViz),
		newRoute("POST", "/ks/([a-zA-Z0-9_]+)/run[/]*", h.ksStartRun),
		newRoute("OPTIONS", "/ks/([a-zA-Z0-9_]+)/run[/]*", h.ksStartRunOptions),
		newRoute("DELETE", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)[/]*", h.ksStopRun),
		newRoute("OPTIONS", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)[/]*", h.ksStopRunOptions),
		newRoute("DELETE", "/ks/([a-zA-Z0-9_]+)[/]*", h.ksDrop),
		newRoute("OPTIONS", "/ks/([a-zA-Z0-9_]+)[/]*", h.ksDropOptions),
	}

	mux.Handle("/", h)

	sc.ScriptDefCache = sc.NewScriptDefCache()
	api.NodeDependencyReadynessCache = api.NewNodeDependencyReadynessCache()

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

	logger.Info("listening on %d...", h.Env.Webapi.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", h.Env.Webapi.Port), mux); err != nil {
		log.Fatalf("%s", err.Error())
	}
}
