package monitoring

import (
	"fmt"
	"strconv"
	"strings"
)

type Stats struct {
	CPUPercent float64
	MemoryPercent float64
	DiskPercent float64
	NetworkRxBytes uint64
	NetworkTxBytes uint64
	NetworkRxBytesPerSec float64
	NetworkTxBytesPerSec float64
}

func Parse(raw string) (Stats, error) {
	var s Stats
	seen := 0
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) != 2 { continue }
		v := strings.TrimSpace(parts[1])
		switch parts[0] {
		case "cpu": s.CPUPercent, _ = strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64); seen++
		case "mem": s.MemoryPercent, _ = strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64); seen++
		case "disk": s.DiskPercent, _ = strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64); seen++
		case "net_rx": s.NetworkRxBytes, _ = strconv.ParseUint(v, 10, 64); seen++
		case "net_tx": s.NetworkTxBytes, _ = strconv.ParseUint(v, 10, 64); seen++
		}
	}
	if seen == 0 { return Stats{}, fmt.Errorf("no system statistics returned") }
	return s, nil
}

func Command() string {
	return "sh -lc 'cpu=$(top -bn1 2>/dev/null | awk \"/Cpu\\(s\\)/{print 100-$8; exit}\"); mem=$(free 2>/dev/null | awk \"/Mem:/{printf \\\"%.1f\\\",100*$3/$2}\"); disk=$(df -P / | awk \"NR==2{gsub(/%/,\\\"\\\",$5);print $5}\"); net=$(awk \"NR>2{rx+=$2;tx+=$10}END{print rx,tx}\" /proc/net/dev 2>/dev/null); set -- $net; printf \"cpu=%s\\nmem=%s\\ndisk=%s\\nnet_rx=%s\\nnet_tx=%s\\n\" \"$cpu\" \"$mem\" \"$disk\" \"$1\" \"$2\"'"
}
