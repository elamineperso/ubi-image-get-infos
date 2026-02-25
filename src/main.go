package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	ListenPort = "8080"
)

// Global variables injected via environment variables
var (
	podName      string
	podNamespace string
	podIP        string
	nodeName     string
	nodeIP       string
	zone         string
	region       string
)

func main() {
	nodeName = os.Getenv("NODE_NAME")
	podName = os.Getenv("POD_NAME")
	podNamespace = os.Getenv("POD_NAMESPACE")
	podIP = os.Getenv("POD_IP")
	nodeIP = os.Getenv("NODE_IP")

	// AZ and Region now injected directly
	zone = os.Getenv("ZONE")
	region = os.Getenv("REGION")

	if zone == "" {
		zone = "ZONE UNKNOWN"
	}

	if region == "" {
		region = "REGION UNKNOWN"
	}

	log.Printf("✅ Pod %s running on node %s (%s)", podName, nodeName, nodeIP)

	http.HandleFunc("/", infoHandler)
	http.HandleFunc("/api/az", azHandler)

	log.Printf("🌍 Web server listening on port %s", ListenPort)
	log.Fatal(http.ListenAndServe(":"+ListenPort, nil))
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	serverTime := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	w.WriteHeader(http.StatusOK)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<title>Pod & Node Info</title>
</head>
<body>
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
		<li><strong>Region:</strong> %s</li>
		<li><strong>Zone:</strong> %s</li>
	</ul>

	<h2>Time</h2>
	<ul>
		<li><strong>Server Time (UTC):</strong> %s</li>
	</ul>
</body>
</html>
`,
		podName,
		podNamespace,
		podIP,
		nodeName,
		nodeIP,
		region,
		zone,
		serverTime,
	)

	fmt.Fprint(w, html)

	log.Printf("Served / | Region=%s Zone=%s ServerTime=%s", region, zone, serverTime)
}

func azHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{ "az": "%s" }`, zone)

	log.Printf("Served /api/az | Zone=%s", zone)
}
