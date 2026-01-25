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
	log.Printf("🌍 Web server listening on port %s", ListenPort)
	log.Fatal(http.ListenAndServe(":"+ListenPort, nil))
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	serverTime := time.Now().UTC()
	serverTimeStr := serverTime.Format("2006-01-02T15:04:05.000Z")

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
			text-align: left;
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
		.delta {
			font-size: 1.2em;
			font-weight: bold;
			color: #d9534f;
		}
		.note {
			margin-top: 15px;
			padding: 12px;
			background: #fff3cd;
			border-left: 4px solid #f0ad4e;
			font-size: 0.95em;
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
<h2>Timing</h2>
		<ul>
			<li>
				<strong>Server Time (UTC):</strong><br>
				<span id="serverTime" data-utc="%s">%s</span>
			</li>
			<li>
				<strong>Client Desktop Time:</strong><br>
				<span id="clientTime">loading...</span>
			</li>
			<li>
				<strong>Latency Delta*</strong><br>
				<span class="delta" id="latencyDelta">calculating...</span>
			</li>
		</ul>

		<div class="note">
			⚠️ <strong>This delta is NOT network RTT.</strong>
			<ul>
				<li>Network latency</li>
				<li>Browser rendering delay</li>
				<li>JavaScript execution delay</li>
				<li>Clock skew between machines</li>
			</ul>
		</div>


		
	</div>

	<script>
		const clientTime = new Date();
		document.getElementById("clientTime").innerText =
			clientTime.toISOString();

		const serverTimeStr =
			document.getElementById("serverTime").dataset.utc;
		const serverTime = new Date(serverTimeStr);

		const deltaMs = clientTime - serverTime;
		document.getElementById("latencyDelta").innerText =
			deltaMs + " ms";
	</script>
</body>
</html>
`,
		serverTimeStr,
		serverTimeStr,
		podName,
		podNamespace,
		podIP,
		nodeName,
		nodeIP,
		region,
		zone,
	)

	fmt.Fprint(w, html)

	log.Printf(
		"Served request | Region=%s Zone=%s ServerTime=%s",
		region, zone, serverTimeStr,
	)
}
