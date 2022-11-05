package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/capillariesio/capillaries/pkg/api"
	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfdb"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
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
	l.Error("cannot process %s: %s", urlPath, err.Error())
	respJson, err := json.Marshal(ApiResponse{Error: ApiResponseError{Msg: err.Error()}})
	if err != nil {
		http.Error(w, fmt.Sprintf("unexpected: cannot serialize error response %s", err.Error()), httpStatus)
	} else {
		http.Error(w, string(respJson), httpStatus)
	}
}

func WriteApiSuccess(l *l.Logger, w http.ResponseWriter, data interface{}) {
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
	cqlSession, err := cql.NewSession(h.Env, "")
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
	Props   *wfmodel.RunAffectedNodes `json:"props"`
	History []*wfmodel.RunHistory     `json:"history"`
}

func (h *UrlHandler) ksRuns(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	// Get all runs that were ever registered (maybe even not started)
	allRunsProps, err := wfdb.GetAllRunsProperties(cqlSession, keyspace)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// Make a ref map RunId->FullRunInfo
	resultMap := map[int16]*FullRunInfo{}
	for _, runProps := range allRunsProps {
		resultMap[runProps.RunId] = &FullRunInfo{Props: runProps}
	}

	// Get run history
	runs, err := api.GetRunHistory(h.L, cqlSession, keyspace)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	// Enrich result with history
	for _, run := range runs {
		fullRunInfo := resultMap[run.RunId]
		fullRunInfo.History = append(fullRunInfo.History, run)
	}

	// Map to list
	result := make([]*FullRunInfo, len(resultMap))
	for idx, runProps := range allRunsProps {
		result[idx] = resultMap[runProps.RunId]
	}

	WriteApiSuccess(h.L, w, result)
}

func (h *UrlHandler) ksNodes(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	result, err := api.GetRunsNodeHistory(h.L, cqlSession, keyspace, []int16{})
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	WriteApiSuccess(h.L, w, result)
}

func (h *UrlHandler) ksRunNodes(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace)
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

	result, err := api.GetRunsNodeHistory(h.L, cqlSession, keyspace, []int16{int16(runId)})
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	WriteApiSuccess(h.L, w, result)
}

func (h *UrlHandler) ksRunNodeBatches(w http.ResponseWriter, r *http.Request) {
	keyspace := getField(r, 0)
	cqlSession, err := cql.NewSession(h.Env, keyspace)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	defer cqlSession.Close()

	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	runId, err := strconv.Atoi(getField(r, 1))
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}

	nodeName := getField(r, 2)
	result, err := wfdb.GetRunNodeBatchHistory(h.L, cqlSession, keyspace, int16(runId), nodeName)
	if err != nil {
		WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
		return
	}
	WriteApiSuccess(h.L, w, result)
}

func (h *UrlHandler) ksStartRun(w http.ResponseWriter, r *http.Request) {
	// keyspace := getField(r, 0)
	// cqlSession, err := cql.NewSession(h.Env, keyspace)
	// if err != nil {
	// 	WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
	// 	return
	// }

	// TODO:
	// - open ampq
	// - get scriptfile, params from the body
	//api.StartRun(h.Env, h.L, ampq, scriptFileUri, scriptParamsUri, cqlSession, keyspace, startNodes)
}

func (h *UrlHandler) ksStopRun(w http.ResponseWriter, r *http.Request) {
	// keyspace := getField(r, 0)
	// cqlSession, err := cql.NewSession(h.Env, keyspace)
	// if err != nil {
	// 	WriteApiError(h.L, w, r.URL.Path, err, http.StatusInternalServerError)
	// 	return
	// }

	// runId := getField(r, 1)

	// TODO:
	// - get comment from the body
	//api.StopRun(h.L, cqlSession, keyspace, runId, comment)
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
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run[/]*", h.ksRuns),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/node[/]*", h.ksNodes),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)/node[/]*", h.ksRunNodes),
		newRoute("GET", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)/node/([a-zA-Z0-9_]+)/batch[/]*", h.ksRunNodeBatches),
		newRoute("POST", "/ks/([a-zA-Z0-9_]+)/run[/]*", h.ksStartRun),
		newRoute("DELETE", "/ks/([a-zA-Z0-9_]+)/run/([0-9]+)[/]*", h.ksStopRun),
	}

	mux.Handle("/", h)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}
}
