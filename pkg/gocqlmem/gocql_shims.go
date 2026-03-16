package gocqlmem

import (
	"context"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// Ideally, these interface should have been defined by gocql

type Iter interface {
	Host() *gocql.HostInfo
	Columns() []gocql.ColumnInfo
	Attempts() int
	Latency() int64
	Keyspace() string
	Table() string
	Scanner() gocql.Scanner
	Scan(dest ...interface{}) bool
	GetCustomPayload() map[string][]byte
	Warnings() []string
	Close() error
	WillSwitchPage() bool
	PageState() []byte
	NumRows() int
	RowData() (gocql.RowData, error)
	SliceMap() ([]map[string]interface{}, error)
	MapScan(m map[string]interface{}) bool

	// These methods are not in gocql, but it's worth adding them:
	Err() error       // Called by tests and Query wrappers around Iter (why do we need them, btw?): MapScan, Scan, ScanCAS, MapScanCAS
	SetErr(err error) // Called by Iter.Scan
}

type gocqlIter struct {
	i *gocql.Iter
}

func (iter *gocqlIter) Host() *gocql.HostInfo {
	return iter.i.Host()
}

func (iter *gocqlIter) Columns() []gocql.ColumnInfo {
	return iter.i.Columns()
}

func (iter *gocqlIter) Attempts() int {
	return iter.i.Attempts()
}

func (iter *gocqlIter) Latency() int64 {
	return iter.i.Latency()
}

func (iter *gocqlIter) Keyspace() string {
	return iter.i.Keyspace()
}

func (iter *gocqlIter) Table() string {
	return iter.i.Table()
}

func (iter *gocqlIter) Scanner() gocql.Scanner {
	return iter.i.Scanner()
}

func (iter *gocqlIter) Scan(dest ...interface{}) bool {
	return iter.i.Scan(dest...)
}

func (iter *gocqlIter) GetCustomPayload() map[string][]byte {
	return iter.i.GetCustomPayload()
}
func (iter *gocqlIter) Warnings() []string {
	return iter.i.Warnings()
}

func (iter *gocqlIter) Close() error {
	return iter.i.Close()
}

func (iter *gocqlIter) WillSwitchPage() bool {
	return iter.i.WillSwitchPage()
}

func (iter *gocqlIter) PageState() []byte {
	return iter.i.PageState()
}

func (iter *gocqlIter) NumRows() int {
	return iter.i.NumRows()
}

func (iter *gocqlIter) RowData() (gocql.RowData, error) {
	return iter.i.RowData()
}

func (iter *gocqlIter) SliceMap() ([]map[string]interface{}, error) {
	return iter.i.SliceMap()
}

func (iter *gocqlIter) MapScan(m map[string]interface{}) bool {
	return iter.i.MapScan(m)
}

func (iter *gocqlIter) Err() error {
	return nil
}

func (iter *gocqlIter) SetErr(err error) {
}

type Query interface {
	Consistency(c gocql.Consistency) Query
	GetConsistency() gocql.Consistency
	CustomPayload(customPayload map[string][]byte) Query
	Trace(trace gocql.Tracer) Query
	Observer(observer gocql.QueryObserver) Query
	PageSize(n int) Query
	DefaultTimestamp(enable bool) Query
	WithTimestamp(timestamp int64) Query
	RoutingKey(routingKey []byte) Query
	Keyspace() string
	Prefetch(p float64) Query
	RetryPolicy(r gocql.RetryPolicy) Query
	SetSpeculativeExecutionPolicy(sp gocql.SpeculativeExecutionPolicy) Query
	IsIdempotent() bool
	Idempotent(value bool) Query
	Bind(v ...interface{}) Query
	SerialConsistency(cons gocql.Consistency) Query
	PageState(state []byte) Query
	NoSkipMetadata() Query
	Exec() error
	ExecContext(ctx context.Context) error
	Iter() Iter
	IterContext(ctx context.Context) Iter
	MapScan(m map[string]interface{}) error
	MapScanContext(ctx context.Context, m map[string]interface{}) error
	Scan(dest ...interface{}) error
	ScanContext(ctx context.Context, dest ...interface{}) error
	ScanCAS(dest ...interface{}) (applied bool, err error)
	ScanCASContext(ctx context.Context, dest ...interface{}) (applied bool, err error)
	MapScanCAS(dest map[string]interface{}) (applied bool, err error)
	MapScanCASContext(ctx context.Context, dest map[string]interface{}) (applied bool, err error)
	SetHostID(hostID string) Query
	GetHostID() string
	SetKeyspace(keyspace string) Query
	WithNowInSeconds(now int) Query
}

type gocqlQuery struct {
	q *gocql.Query
}

func (q *gocqlQuery) Consistency(c gocql.Consistency) Query {
	q.q.Consistency(c)
	return q
}

func (q *gocqlQuery) GetConsistency() gocql.Consistency {
	return q.q.GetConsistency()
}

func (q *gocqlQuery) CustomPayload(customPayload map[string][]byte) Query {
	q.q.CustomPayload(customPayload)
	return q
}

func (q *gocqlQuery) Trace(trace gocql.Tracer) Query {
	q.q.Trace(trace)
	return q
}

func (q *gocqlQuery) Observer(observer gocql.QueryObserver) Query {
	q.q.Observer(observer)
	return q
}
func (q *gocqlQuery) PageSize(n int) Query {
	q.q.PageSize(n)
	return q
}
func (q *gocqlQuery) DefaultTimestamp(enable bool) Query {
	q.q.DefaultTimestamp(enable)
	return q
}
func (q *gocqlQuery) WithTimestamp(timestamp int64) Query {
	q.q.WithTimestamp(timestamp)
	return q
}
func (q *gocqlQuery) RoutingKey(routingKey []byte) Query {
	q.q.RoutingKey(routingKey)
	return q
}
func (q *gocqlQuery) Keyspace() string {
	return q.q.Keyspace()
}
func (q *gocqlQuery) Prefetch(p float64) Query {
	q.q.Prefetch(p)
	return q
}
func (q *gocqlQuery) RetryPolicy(r gocql.RetryPolicy) Query {
	q.q.RetryPolicy(r)
	return q
}
func (q *gocqlQuery) SetSpeculativeExecutionPolicy(sp gocql.SpeculativeExecutionPolicy) Query {
	q.q.SetSpeculativeExecutionPolicy(sp)
	return q
}
func (q *gocqlQuery) IsIdempotent() bool {
	return q.q.IsIdempotent()
}
func (q *gocqlQuery) Idempotent(value bool) Query {
	q.q.Idempotent(value)
	return q
}
func (q *gocqlQuery) Bind(v ...interface{}) Query {
	q.q.Bind(v...)
	return q
}
func (q *gocqlQuery) SerialConsistency(cons gocql.Consistency) Query {
	q.q.SerialConsistency(cons)
	return q
}
func (q *gocqlQuery) PageState(state []byte) Query {
	q.q.PageState(state)
	return q
}
func (q *gocqlQuery) NoSkipMetadata() Query {
	q.q.NoSkipMetadata()
	return q
}
func (q *gocqlQuery) Exec() error {
	return q.q.Exec()
}
func (q *gocqlQuery) ExecContext(ctx context.Context) error {
	return q.q.ExecContext(ctx)
}
func (q *gocqlQuery) Iter() Iter {
	return &gocqlIter{i: q.q.Iter()}
}
func (q *gocqlQuery) IterContext(ctx context.Context) Iter {
	return &gocqlIter{i: q.q.IterContext(ctx)}
}
func (q *gocqlQuery) MapScan(m map[string]interface{}) error {
	return q.q.MapScan(m)
}
func (q *gocqlQuery) MapScanContext(ctx context.Context, m map[string]interface{}) error {
	return q.q.MapScanContext(ctx, m)
}
func (q *gocqlQuery) Scan(dest ...interface{}) error {
	return q.q.Scan(dest...)
}
func (q *gocqlQuery) ScanContext(ctx context.Context, dest ...interface{}) error {
	return q.q.ScanContext(ctx, dest...)
}
func (q *gocqlQuery) ScanCAS(dest ...interface{}) (applied bool, err error) {
	return q.q.ScanCAS(dest...)
}
func (q *gocqlQuery) ScanCASContext(ctx context.Context, dest ...interface{}) (applied bool, err error) {
	return q.q.ScanCASContext(ctx, dest...)
}
func (q *gocqlQuery) MapScanCAS(dest map[string]interface{}) (applied bool, err error) {
	return q.q.MapScanCAS(dest)
}
func (q *gocqlQuery) MapScanCASContext(ctx context.Context, dest map[string]interface{}) (applied bool, err error) {
	return q.q.MapScanCASContext(ctx, dest)
}
func (q *gocqlQuery) SetHostID(hostID string) Query {
	q.q.SetHostID(hostID)
	return q
}
func (q *gocqlQuery) GetHostID() string {
	return q.q.GetHostID()
}
func (q *gocqlQuery) SetKeyspace(keyspace string) Query {
	q.q.SetKeyspace(keyspace)
	return q
}
func (q *gocqlQuery) WithNowInSeconds(now int) Query {
	q.q.WithNowInSeconds(now)
	return q
}

type Session interface {
	AwaitSchemaAgreement(ctx context.Context) error
	Query(stmt string, values ...interface{}) Query
	Bind(stmt string, b func(q *gocql.QueryInfo) ([]interface{}, error)) Query
	Close()
	Closed() bool
	KeyspaceMetadata(keyspace string) (*gocql.KeyspaceMetadata, error)
	Batch(typ gocql.BatchType) *gocql.Batch
	GetHosts() []*gocql.HostInfo
}

type gocqlSession struct {
	s *gocql.Session
}

// This helper was made public intentionally, this is how users instantiate sessions
func NewGocqlSession(s *gocql.Session) Session {
	return &gocqlSession{
		s: s,
	}
}

func (s *gocqlSession) AwaitSchemaAgreement(ctx context.Context) error {
	return s.s.AwaitSchemaAgreement(ctx)
}

func (s *gocqlSession) Query(stmt string, values ...interface{}) Query {
	return &gocqlQuery{q: s.s.Query(stmt, values...)}
}

func (s *gocqlSession) Bind(stmt string, b func(q *gocql.QueryInfo) ([]interface{}, error)) Query {
	return &gocqlQuery{q: s.s.Bind(stmt, b)}
}

func (s *gocqlSession) Close() {
	s.s.Close()
}

func (s *gocqlSession) Closed() bool {
	return s.s.Closed()
}

func (s *gocqlSession) KeyspaceMetadata(keyspace string) (*gocql.KeyspaceMetadata, error) {
	return s.s.KeyspaceMetadata(keyspace)
}

func (s *gocqlSession) Batch(typ gocql.BatchType) *gocql.Batch {
	return s.s.Batch(typ)
}

func (s *gocqlSession) GetHosts() []*gocql.HostInfo {
	return s.s.GetHosts()
}
