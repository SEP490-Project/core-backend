package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"
)

type SystemService struct {
	config    *config.AppConfig
	startTime time.Time
}

func NewSystemService(config *config.AppConfig) iservice.SystemService {
	return &SystemService{
		config:    config,
		startTime: time.Now(),
	}
}

func (s *SystemService) GetSystemSpecs(ctx context.Context) (*responses.SystemSpecsResponse, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	hostname, _ := os.Hostname()
	ips := getIPs()
	disk := getDiskUsage("/")

	return &responses.SystemSpecsResponse{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
		GoVersion:  runtime.Version(),
		Goroutines: runtime.NumGoroutine(),
		Memory: responses.MemoryStats{
			Alloc:        formatBytes(m.Alloc),
			TotalAlloc:   formatBytes(m.TotalAlloc),
			Sys:          formatBytes(m.Sys),
			NumGC:        m.NumGC,
			PauseTotalNs: time.Duration(m.PauseTotalNs).String(),
		},
		Uptime:      time.Since(s.startTime).String(),
		CurrentTime: time.Now(),
		Timezone:    time.Now().Location().String(),
		Environment: s.config.Server.Environment,
		BuildInfo: responses.BuildInfo{
			Version:   "1.0.0",    // TODO: Inject via ldflags
			Commit:    "HEAD",     // TODO: Inject via ldflags
			BuildTime: time.Now(), // TODO: Inject via ldflags
		},
		Network: responses.NetworkInfo{
			HostName: hostname,
			IPs:      ips,
		},
		Kubernetes: responses.KubernetesInfo{
			PodName:   getEnv("K8S_POD_NAME", "unknown"),
			Namespace: getEnv("K8S_NAMESPACE", "unknown"),
			NodeName:  getEnv("K8S_NODE_NAME", "unknown"),
			PodIP:     getEnv("K8S_POD_IP", "unknown"),
		},
		Disk: disk,
	}, nil
}

func getIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getDiskUsage(path string) responses.DiskInfo {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return responses.DiskInfo{Total: "N/A", Free: "N/A", Used: "N/A"}
	}

	// Available blocks * size per block = available space in bytes
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free

	return responses.DiskInfo{
		Total: formatBytes(total),
		Free:  formatBytes(free),
		Used:  formatBytes(used),
	}
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
