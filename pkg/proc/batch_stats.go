package proc

import (
	"fmt"
	"strings"
	"time"
)

// Time stats int64 in nanos, float64 seconds
type BatchStats struct {
	Src              string
	Dst              string
	RowsRead         int
	RowsWritten      int
	Elapsed          time.Duration
	DataCount        int64
	DataElapsedMin   int64
	DataElapsedMax   int64
	DataElapsedTotal int64
	DataElapsedAvg   float64
	IdxCount         int64
	IdxElapsedMin    int64
	IdxElapsedMax    int64
	IdxElapsedTotal  int64
	IdxElapsedAvg    float64
}

func (bs *BatchStats) UpdateElapsedStats(dur time.Duration, instr *TableInserter) {
	bs.Elapsed = dur
	if instr != nil {
		bs.DataCount, bs.DataElapsedMin, bs.DataElapsedMax, bs.DataElapsedTotal, bs.DataElapsedAvg = instr.DataStats.GetStats()
		bs.IdxCount, bs.IdxElapsedMin, bs.IdxElapsedMax, bs.IdxElapsedTotal, bs.IdxElapsedAvg = instr.IdxStats.GetStats()
	}
}

// TODO: this is what we display in WebUI, make it structured
func (bs *BatchStats) ToString() string {
	s := bs.Elapsed.Seconds()
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "inserter_elapsed: %.4f s; %s -> %s; row_reads: %d, %.1f r/s; ", s, bs.Src, bs.Dst, bs.RowsRead, float64(bs.RowsRead)/s)
	// Display even zero writes when there is at least one read
	if bs.RowsRead > 0 {
		fmt.Fprintf(&sb, "row_writes: %d, %.1f w/s; ", bs.RowsWritten, float64(bs.RowsWritten)/s)
	}
	if bs.DataCount > 0 {
		fmt.Fprintf(&sb, "data_inserts: %d, min/avg/max %.4f / %.4f / %.4f s, total %.4f s; ", bs.DataCount, float64(bs.DataElapsedMin)/1000000000.0, bs.DataElapsedAvg, float64(bs.DataElapsedMax)/1000000000.0, float64(bs.DataElapsedTotal)/1000000000.0)
	}
	if bs.IdxCount > 0 {
		fmt.Fprintf(&sb, "idx_inserts: %d, min/avg/max %.4f / %.4f / %.4f s, total %.4f s", bs.IdxCount, float64(bs.IdxElapsedMin)/1000000000.0, bs.IdxElapsedAvg, float64(bs.IdxElapsedMax)/1000000000.0, float64(bs.IdxElapsedTotal)/1000000000.0)
	}
	return sb.String()
}
