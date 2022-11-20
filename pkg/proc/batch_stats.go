package proc

import (
	"fmt"
	"time"
)

type BatchStats struct {
	RowsRead    int
	RowsWritten int
	Elapsed     time.Duration
}

func (bs *BatchStats) ToString() string {
	s := bs.Elapsed.Seconds()
	return fmt.Sprintf("{read: %d, written: %d, elapsed:%.3f, r/s: %.1f, w/s: %.1f}", bs.RowsRead, bs.RowsWritten, s, float64(bs.RowsRead)/s, float64(bs.RowsWritten)/s)
}
