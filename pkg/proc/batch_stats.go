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
	fmt.Fprintf(&sb, " ", s, bs.RowsRead, float64(bs.RowsRead)/s, bs.Src, bs.Dst)
	if bs.RowsRead > 0 {
		fmt.Fprintf(&sb, "row writes: %d, w/s: %.1f, ", bs.RowsWritten, float64(bs.RowsWritten)/s)
	}
	if bs.DataCount > 0 {
		fmt.Fprintf(&sb, "data inserts: %d, min/avg/max: %.4fs/%.4fs/%.4fs, total: %.4fs, ", bs.DataCount, float64(bs.DataElapsedMin)/1000000000.0, bs.DataElapsedAvg, float64(bs.DataElapsedMax)/1000000000.0, float64(bs.DataElapsedTotal)/1000000000.0)
	}
	if bs.IdxCount > 0 {
		fmt.Fprintf(&sb, "idx inserts: %d, min/avg/max: %.4fs/%.4fs/%.4fs, total: %.4fs", bs.IdxCount, float64(bs.IdxElapsedMin)/1000000000.0, bs.IdxElapsedAvg, float64(bs.IdxElapsedMax)/1000000000.0, float64(bs.IdxElapsedTotal)/1000000000.0)
	}
	return sb.String()
}
