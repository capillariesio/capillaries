package api

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/gocqlshims"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

const ProhibitedKeyspaceNameRegex = "^system"
const AllowedKeyspaceNameRegex = "[a-zA-Z0-9_]+"

// Used by Webapi to ignore Cassandra system keyspaces
func IsSystemKeyspaceName(keyspace string) bool {
	re := regexp.MustCompile(ProhibitedKeyspaceNameRegex)
	invalidNamePieceFound := re.FindString(keyspace)
	return len(invalidNamePieceFound) > 0
}

func checkKeyspaceNameAllowed(keyspace string) error {
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

// Used by Toolbelt get_table_cql cmd, prints out CQL to create all workflow tables and data/index tables for a specific keyspace.
// startNodeNames - list of script nodes that will be started immediately upon run start,
// helps find out which tables referenced by the script will be affected, to avoid unnecessary table creation.
func GetTablesCql(script *sc.ScriptDef, keyspace string, runId int16, startNodeNames []string) string {
	sb := strings.Builder{}
	sb.WriteString("-- Workflow\n")
	fmt.Fprintf(&sb, "%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.BatchHistoryEvent{}), keyspace, wfmodel.TableNameBatchHistory))
	fmt.Fprintf(&sb, "%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.NodeHistoryEvent{}), keyspace, wfmodel.TableNameNodeHistory))
	fmt.Fprintf(&sb, "%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.RunHistoryEvent{}), keyspace, wfmodel.TableNameRunHistory))
	fmt.Fprintf(&sb, "%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.RunProperties{}), keyspace, wfmodel.TableNameRunAffectedNodes))
	fmt.Fprintf(&sb, "%s\n", wfmodel.GetCreateTableCql(reflect.TypeOf(wfmodel.RunCounter{}), keyspace, wfmodel.TableNameRunCounter))
	qb := cql.QueryBuilder{}
	fmt.Fprintf(&sb, "%s\n", qb.Keyspace(keyspace).Write("ks", keyspace).Write("last_run", 0).InsertUnpreparedQuery(wfmodel.TableNameRunCounter, cql.IgnoreIfExists))

	for _, nodeName := range script.GetAffectedNodes(startNodeNames) {
		node, ok := script.ScriptNodes[nodeName]
		if !ok || !node.HasTableCreator() {
			continue
		}
		fmt.Fprintf(&sb, "-- %s\n", nodeName)
		fmt.Fprintf(&sb, "%s\n", proc.CreateDataTableCql(keyspace, runId, &node.TableCreator))
		for idxName, idxDef := range node.TableCreator.Indexes {
			fmt.Fprintf(&sb, "%s\n", proc.CreateIdxTableCql(keyspace, runId, idxName, idxDef, &node.TableCreator))
		}
	}
	return sb.String()
}

// Used by Toolbelt and Webapi to drop Cassandra keyspace
func DropKeyspace(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string) error {
	logger.PushF("api.DropKeyspace")
	defer logger.PopF()

	dbStartTime := time.Now()

	if err := checkKeyspaceNameAllowed(keyspace); err != nil {
		return err
	}

	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(keyspace).
		DropKeyspace()
	if err := cqlSession.Query(q).Exec(); err != nil {
		return db.WrapDbErrorWithQuery("cannot drop keyspace", q, err)
	}

	if err := db.VerifyKeyspaceDeleted(cqlSession, keyspace); err != nil {
		return err
	}

	logger.Info("drop keyspace %s took %.2fs", keyspace, time.Since(dbStartTime).Seconds())

	return nil
}
