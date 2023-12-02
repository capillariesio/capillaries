package wfmodel

const TableNameRunCounter = "wf_run_counter"

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type RunCounter struct {
	Keyspace int `header:"ks" format:"%20s" column:"ks" type:"text" key:"true"`
	LastRun  int `header:"lr" format:"%3d" column:"last_run" type:"int"`
}
