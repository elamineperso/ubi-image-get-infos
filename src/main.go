package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	ListenPort     = "8080"
	ZoneLabelKey   = "topology.kubernetes.io/zone"
	RegionLabelKey = "topology.kubernetes.io/region"
)

// Global variables injected via Downward API
var (
	podName      string
	podNamespace string
	podIP        string
	nodeName     string
	nodeIP       string

	clientset *kubernetes.Clientset

	refreshInterval time.Duration
	apiTimeout      time.Duration
	accessLog       bool

	nodeMeta = &nodeMetadata{
		Zone:      "ZONE UNKNOWN",
		Region:    "REGION UNKNOWN",
		NodeIP:    "0.0.0.0",
		LastError: "not initialized",
	}
)

type nodeMetadata struct {
	mu         sync.RWMutex
	Zone       string
	Region     string
	NodeIP     string
	LastUpdate time.Time
	LastError  string
}

func (m *nodeMetadata) set(zone, region, nodeIP, lastError string, lastUpdate time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Zone = zone
	m.Region = region
	m.NodeIP = nodeIP
	m.LastError = lastError
	m.LastUpdate = lastUpdate
}

func (m *nodeMetadata) snapshot() nodeMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return nodeMetadata{
		Zone:       m.Zone,
		Region:     m.Region,
		NodeIP:     m.NodeIP,
		LastUpdate: m.LastUpdate,
		LastError:  m.LastError,
	}
}

func main() {
	nodeName = os.Getenv("NODE_NAME")
	podName = os.Getenv("POD_NAME")
	podNamespace = os.Getenv("POD_NAMESPACE")
	podIP = os.Getenv("POD_IP")
	nodeIP = os.Getenv("NODE_IP")
	refreshInterval = getEnvDuration("AZ_REFRESH_INTERVAL", 60*time.Second)
	apiTimeout = getEnvDuration("KUBE_API_TIMEOUT", 2*time.Second)
	accessLog = getEnvBool("ACCESS_LOG", false)

	if nodeName == "" {
		log.Fatal("ERROR: NODE_NAME environment variable not set")
	}

	log.Printf("Pod %s running on node %s (%s)", podName, nodeName, nodeIP)

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("ERROR: failed to get in-cluster config: %v", err)
	}
	config.QPS = float32(getEnvFloat("KUBE_CLIENT_QPS", 20))
	config.Burst = getEnvInt("KUBE_CLIENT_BURST", 40)

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("ERROR: failed to create Kubernetes client: %v", err)
	}

	// Warm-up cache before serving traffic.
	refreshNodeMetadata()
	go refreshLoop()

	http.HandleFunc("/", infoHandler)
	http.HandleFunc("/api/az", azHandler)
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/readyz", readyzHandler)

	log.Printf("Web server listening on port %s (refresh=%s)", ListenPort, refreshInterval)
	srv := &http.Server{
		Addr:              ":" + ListenPort,
		Handler:           nil,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	serverTime := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	meta := nodeMeta.snapshot()

	w.WriteHeader(http.StatusOK)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<title>OpenShift Pod & Node Info</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			background-color: #f5f7fa;
			margin: 0;
			padding: 0;
		}
		.container {
			max-width: 720px;
			margin: 60px auto;
			background: #ffffff;
			padding: 30px 40px;
			border-radius: 8px;
			box-shadow: 0 4px 10px rgba(0,0,0,0.1);
		}
		h1, h2 {
			margin-bottom: 10px;
		}
		ul {
			list-style: none;
			padding: 0;
		}
		li {
			margin: 6px 0;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>Pod & Node Information</h1>

		<h2>Pod</h2>
		<ul>
			<li><strong>Name:</strong> %s</li>
			<li><strong>Namespace:</strong> %s</li>
			<li><strong>IP:</strong> %s</li>
		</ul>

			<h2>Node</h2>
			<ul>
				<li><strong>Name:</strong> %s</li>
				<li><strong>IP:</strong> %s</li>
				<li><strong>Region:</strong> <span style="color: green;">%s</span></li>
				<li><strong>Zone:</strong> <span style="color: blue;">%s</span></li>
			</ul>

			<h2>Time</h2>
			<ul>
				<li><strong>Server Time (UTC):</strong> <span id="serverTime">%s</span></li>
				<li><strong>Client Time:</strong> <span id="clientTime">loading...</span></li>
				<li><strong>Last AZ Refresh:</strong> %s</li>
				<li><strong>Last AZ Error:</strong> %s</li>
			</ul>
		</div>

	<script>
		document.getElementById("clientTime").innerText =
			new Date().toISOString();
	</script>
</body>
</html>
`,
		podName,
		podNamespace,
		podIP,
		nodeName,
		meta.NodeIP,
		meta.Region,
		meta.Zone,
		serverTime,
		formatTime(meta.LastUpdate),
		emptyAsDash(meta.LastError),
	)

	fmt.Fprint(w, html)

	if accessLog {
		log.Printf("Served / | Region=%s Zone=%s ServerTime=%s", meta.Region, meta.Zone, serverTime)
	}
}

func azHandler(w http.ResponseWriter, r *http.Request) {
	meta := nodeMeta.snapshot()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"az":          meta.Zone,
		"region":      meta.Region,
		"node_name":   nodeName,
		"node_ip":     meta.NodeIP,
		"pod_name":    podName,
		"pod_ip":      podIP,
		"updated_at":  formatTime(meta.LastUpdate),
		"last_error":  emptyAsDash(meta.LastError),
		"source":      "in-memory-cache",
		"api_timeout": apiTimeout.String(),
	})

	if accessLog {
		log.Printf("Served /api/az | Zone=%s Region=%s", meta.Zone, meta.Region)
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func readyzHandler(w http.ResponseWriter, r *http.Request) {
	meta := nodeMeta.snapshot()
	if meta.Zone == "" || meta.Zone == "ZONE UNKNOWN" {
		http.Error(w, "zone not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready\n"))
}

func refreshLoop() {
	t := time.NewTicker(refreshInterval)
	defer t.Stop()
	for range t.C {
		refreshNodeMetadata()
	}
}

func refreshNodeMetadata() {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		s := nodeMeta.snapshot()
		nodeMeta.set(s.Zone, s.Region, s.NodeIP, err.Error(), s.LastUpdate)
		log.Printf("AZ refresh failed for node %s: %v", nodeName, err)
		return
	}

	zone := node.Labels[ZoneLabelKey]
	if zone == "" {
		zone = "ZONE UNKNOWN"
	}
	region := node.Labels[RegionLabelKey]
	if region == "" {
		region = "REGION UNKNOWN"
	}
	resolvedNodeIP := findNodeIP(node)
	if resolvedNodeIP == "" && nodeIP != "" {
		resolvedNodeIP = nodeIP
	}
	if resolvedNodeIP == "" {
		resolvedNodeIP = "0.0.0.0"
	}

	nodeMeta.set(zone, region, resolvedNodeIP, "", time.Now().UTC())
}

func findNodeIP(node *corev1.Node) string {
	var external string
	for _, a := range node.Status.Addresses {
		switch a.Type {
		case corev1.NodeInternalIP:
			return a.Address
		case corev1.NodeExternalIP:
			external = a.Address
		}
	}
	return external
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

func emptyAsDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}
