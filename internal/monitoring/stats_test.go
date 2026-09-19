package monitoring

import "testing"

func TestParse(t *testing.T) {
	s, err := Parse("cpu=12.5\nmem=48.2\ndisk=71\nnet_rx=1000\nnet_tx=2000\n")
	if err != nil { t.Fatal(err) }
	if s.CPUPercent != 12.5 || s.MemoryPercent != 48.2 || s.DiskPercent != 71 || s.NetworkRxBytes != 1000 || s.NetworkTxBytes != 2000 { t.Fatalf("unexpected stats: %+v", s) }
}
