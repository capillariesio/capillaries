package sc

type TableReaderDef struct {
	TableName            string `json:"table"`
	ExpectedBatchesTotal int    `json:"expected_batches_total"`
	RowsetSize           int    `json:"rowset_size"` // DefaultRowsetSize = 1000
	TableCreator         *TableCreatorDef
}
