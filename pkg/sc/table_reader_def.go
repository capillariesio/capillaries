package sc

type TableReaderDef struct {
	TableName            string           `json:"table"`
	ExpectedBatchesTotal int              `json:"expected_batches_total,omitempty"`
	RowsetSize           int              `json:"rowset_size,omitempty"` // DefaultRowsetSize = 1000
	TableCreator         *TableCreatorDef `json:"-"`
}
