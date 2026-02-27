package main

import (
"context"
"fmt"
"log"
"net/http"
"os"
"time"

metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
"k8s.io/client-go/kubernetes"
"k8s.io/client-go/rest"
)

const (
ListenPort = "8080"
ZoneLabelKey = "topology.kubernetes.io/zone"
RegionLabelKey = "topology.kubernetes.io/region"
)

// Global variables injected via Downward API
var (
podName string
podNamespace string
podIP string
nodeName string
nodeIP string

clientset *kubernetes.Clientset
)

func main() {
nodeName = os.Getenv("NODE_NAME")
podName = os.Getenv("POD_NAME")
podNamespace = os.Getenv("POD_NAMESPACE")
podIP = os.Getenv("POD_IP")
nodeIP = os.Getenv("NODE_IP")

if nodeName == "" {
log.Fatal("❌ ERROR: NODE_NAME environment variable not set")
}

log.Printf("✅ Pod %s running on node %s (%s)", podName, nodeName, nodeIP)

config, err := rest.InClusterConfig()
if err != nil {
log.Fatalf("❌ ERROR: Failed to get in-cluster config: %v", err)
}

clientset, err = kubernetes.NewForConfig(config)
if err != nil {
log.Fatalf("❌ ERROR: Failed to create Kubernetes client: %v", err)
}

http.HandleFunc("/", infoHandler)
http.HandleFunc("/api/az", azHandler)

log.Printf("🌍 Web server listening on port %s", ListenPort)
log.Fatal(http.ListenAndServe(":"+ListenPort, nil))
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
serverTime := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
if err != nil {
w.WriteHeader(http.StatusInternalServerError)
fmt.Fprintf(w, "Error retrieving node %s: %v", nodeName, err)
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
nodeIP,
region,
zone,
serverTime,
)

fmt.Fprint(w, html)

log.Printf("Served / | Region=%s Zone=%s ServerTime=%s", region, zone, serverTime)
}

func azHandler(w http.ResponseWriter, r *http.Request) {
node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
if err != nil {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusInternalServerError)
fmt.Fprintf(w, `{ "error": "failed to retrieve node" }`)
return
}

zone := node.Labels[ZoneLabelKey]
if zone == "" {
zone = "UNKNOWN"
}

w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
fmt.Fprintf(w, `{ "az": "%s" }`, zone)

log.Printf("Served /api/az | Zone=%s", zone)
}
