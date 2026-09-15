package cql

type IfNotExistsType int

const (
	IfExistsOverwrite IfNotExistsType = 0
	IfNotExistsLwt    IfNotExistsType = 1
)

type QuotePolicyType int

const (
	LeaveQuoteAsIs QuotePolicyType = iota
	ForceUnquote
)

// These errors mimic Cassandra errors, so do not change these strings
const (
	ErrorDoesNotExist                 string = "does not exist"
	ErrorOperationTimedOut            string = "Operation timed out"
	ErrorAmazonKeyspacesZeroResponses string = "Operation failed - received 0 responses and 1 failures" // Saw this from Amazon Keyspaces, slow down
	ErrorSomeSeriousError             string = "some serious Cassandra error"                           // Well, this does not mimic any Cassandra error
	ErrorCannotUpsertDuplicate        string = "dberror:cannot upsert duplicate"
)
