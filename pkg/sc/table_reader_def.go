package sc

type TableReaderDef struct {
	TableName            string           `json:"table" yaml:"table"`
	ExpectedBatchesTotal int              `json:"expected_batches_total,omitempty" yaml:"expected_batches_total,omitempty"`
	RowsetSize           int              `json:"rowset_size,omitempty" yaml:"rowset_size,omitempty"` // DefaultRowsetSize = 1000
	TableCreator         *TableCreatorDef `json:"-"`
}
