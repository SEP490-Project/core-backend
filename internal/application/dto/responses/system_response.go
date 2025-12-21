package responses

import "time"

type SystemSpecsResponse struct {
	OS          string         `json:"os"`
	Arch        string         `json:"arch"`
	NumCPU      int            `json:"num_cpu"`
	GOMAXPROCS  int            `json:"gomaxprocs"`
	GoVersion   string         `json:"go_version"`
	Goroutines  int            `json:"goroutines"`
	Memory      MemoryStats    `json:"memory"`
	CurrentTime time.Time      `json:"current_time"`
	Timezone    string         `json:"timezone"`
	Uptime      string         `json:"uptime"`
	Environment string         `json:"environment"`
	BuildInfo   BuildInfo      `json:"build_info"`
	Network     NetworkInfo    `json:"network"`
	Kubernetes  KubernetesInfo `json:"kubernetes"`
	Disk        DiskInfo       `json:"disk"`
}

type MemoryStats struct {
	Alloc        string `json:"alloc"`
	TotalAlloc   string `json:"total_alloc"`
	Sys          string `json:"sys"`
	NumGC        uint32 `json:"num_gc"`
	PauseTotalNs string `json:"pause_total_ns"`
}

type BuildInfo struct {
	Version   string    `json:"version"`
	Commit    string    `json:"commit"`
	BuildTime time.Time `json:"build_time"`
}

type NetworkInfo struct {
	HostName string   `json:"host_name"`
	IPs      []string `json:"ips"`
}

type KubernetesInfo struct {
	PodName   string `json:"pod_name"`
	NodeName  string `json:"node_name"`
	Namespace string `json:"namespace"`
	PodIP     string `json:"pod_ip"`
}

type DiskInfo struct {
	Total string `json:"total"`
	Free  string `json:"free"`
	Used  string `json:"used"`
}
