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
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

const ProhibitedKeyspaceNameRegex = "^system"

func CheckKeyspaceName(keyspace string) error {
	re := regexp.MustCompile(ProhibitedKeyspaceNameRegex)
	invalidNamePieceFound := re.FindString(keyspace)
	if len(invalidNamePieceFound) > 0 {
		return fmt.Errorf("invalid keyspace name [%s]: prohibited regex is [%s]", keyspace, ProhibitedKeyspaceNameRegex)
	}
	return nil
}

func GetTablesCql(script *sc.ScriptDef, keyspace string, runId int16, startNodeNames []string) string {
	sb := strings.Builder{}
	sb.WriteString("-- Workflow\n")
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.BatchHistory{}), keyspace, wfmodel.TableNameBatchHistory)))
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.NodeHistory{}), keyspace, wfmodel.TableNameNodeHistory)))
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.RunHistory{}), keyspace, wfmodel.TableNameRunHistory)))
	sb.WriteString(fmt.Sprintf("%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.RunAffectedNodes{}), keyspace, wfmodel.TableNameRunAffectedNodes)))
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

func DropKeyspace(logger *l.Logger, cqlSession *gocql.Session, keyspace string) error {
	logger.PushF("DropKeyspace")
	defer logger.PopF()

	if err := CheckKeyspaceName(keyspace); err != nil {
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
