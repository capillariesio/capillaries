package api

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfdb"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

const ProhibitedKeyspaceNameRegex = "^system"
const AllowedKeyspaceNameRegex = "[a-zA-Z0-9_]+"

func IsSystemKeyspaceName(keyspace string) bool {
	re := regexp.MustCompile(ProhibitedKeyspaceNameRegex)
	invalidNamePieceFound := re.FindString(keyspace)
	if len(invalidNamePieceFound) > 0 {
		return true
	}
	return false
}

func checkKeyspaceName(keyspace string) error {
	re := regexp.MustCompile(ProhibitedKeyspaceNameRegex)
	invalidNamePieceFound := re.FindString(keyspace)
	if len(invalidNamePieceFound) > 0 {
		return fmt.Errorf("invalid keyspace name [%s]: prohibited regex is [%s]", keyspace, ProhibitedKeyspaceNameRegex)
	}
	re = regexp.MustCompile(AllowedKeyspaceNameRegex)
	if !re.MatchString(keyspace) {
		return fmt.Errorf("invalid keyspace name [%s]: allowed regex is [%s]", keyspace, AllowedKeyspaceNameRegex)
	}
	return nil
}

// A helper used by Toolbelt get_table_cql cmd, no logging needed
func GetTablesCql(script *sc.ScriptDef, keyspace string, runId int16, startNodeNames []string) string {
	sb := strings.Builder{}
	sb.WriteString("-- Workflow\n")
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.BatchHistoryEvent{}), keyspace, wfmodel.TableNameBatchHistory)))
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.NodeHistoryEvent{}), keyspace, wfmodel.TableNameNodeHistory)))
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.RunHistoryEvent{}), keyspace, wfmodel.TableNameRunHistory)))
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.RunProperties{}), keyspace, wfmodel.TableNameRunAffectedNodes)))
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.RunCounter{}), keyspace, wfmodel.TableNameRunCounter)))
	qb := cql.QueryBuilder{}
	sb.WriteString(fmt.Sprintf("%s\n", qb.Keyspace(keyspace).Write("ks", keyspace).Write("last_run", 0).Insert(wfmodel.TableNameRunCounter, cql.IgnoreIfExists)))

	for _, nodeName := range script.GetAffectedNodes(startNodeNames) {
		node, ok := script.ScriptNodes[nodeName]
		if !ok || !node.HasTableCreator() {
			continue
		}
		sb.WriteString(fmt.Sprintf("-- %s\n", nodeName))
		sb.WriteString(fmt.Sprintf("%s\n", proc.CreateDataTableCql(keyspace, runId, &node.TableCreator)))
		for idxName, idxDef := range node.TableCreator.Indexes {
			sb.WriteString(fmt.Sprintf("%s\n", proc.CreateIdxTableCql(keyspace, runId, idxName, idxDef)))
		}
	}
	return sb.String()
}

// Used by Toolbelt and Webapi
func DropKeyspace(logger *l.Logger, cqlSession *gocql.Session, keyspace string) error {
	logger.PushF("api.DropKeyspace")
	defer logger.PopF()

	if err := checkKeyspaceName(keyspace); err != nil {
		return err
	}

	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(keyspace).
		DropKeyspace()
	if err := cqlSession.Query(q).Exec(); err != nil {
		return cql.WrapDbErrorWithQuery("cannot drop keyspace", q, err)
	}
	return nil
}

// wfdb wrapper for webapi use
func HarvestRunLifespans(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runIds []int16) (wfmodel.RunLifespanMap, error) {
	logger.PushF("api.HarvestRunLifespans")
	defer logger.PopF()

	return wfdb.HarvestRunLifespans(logger, cqlSession, keyspace, runIds)
}

// wfdb wrapper for webapi use
func GetRunProperties(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16) ([]*wfmodel.RunProperties, error) {
	logger.PushF("api.GetRunProperties")
	defer logger.PopF()
	return wfdb.GetRunProperties(logger, cqlSession, keyspace, runId)
}

// wfdb wrapper for webapi use
func GetNodeHistoryForRun(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16) ([]*wfmodel.NodeHistoryEvent, error) {
	logger.PushF("api.GetNodeHistoryForRun")
	defer logger.PopF()

	return wfdb.GetNodeHistoryForRun(logger, cqlSession, keyspace, runId)
}

// wfdb wrapper for webapi use
func GetRunNodeBatchHistory(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16, nodeName string) ([]*wfmodel.BatchHistoryEvent, error) {
	logger.PushF("api.GetRunNodeBatchHistory")
	defer logger.PopF()
	return wfdb.GetRunNodeBatchHistory(logger, cqlSession, keyspace, runId, nodeName)
}
