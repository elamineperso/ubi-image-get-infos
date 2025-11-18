package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	// Kubernetes client libraries
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// The well-known Kubernetes label key for the Availability Zone.
const AzLabelKey = "topology.kubernetes.io/zone"
const ListenPort = "8080"

// Global variables for injected and initialized data
var (
	// Injected via Downward API
	podName      string
	podNamespace string
	podIP        string
	nodeName     string
	nodeIP       string

	// Initialized via InClusterConfig
	clientset *kubernetes.Clientset
)

func main() {
	// --- 1. Initialization Phase (executed once) ---

	// Read all necessary information injected by the Downward API
	nodeName = os.Getenv("NODE_NAME")
	podName = os.Getenv("POD_NAME")
	podNamespace = os.Getenv("POD_NAMESPACE")
	podIP = os.Getenv("POD_IP")
	nodeIP = os.Getenv("NODE_IP")

	// Basic check for required variable
	if nodeName == "" {
		log.Fatal("❌ ERROR: NODE_NAME environment variable not set. Check Downward API configuration.")
	}

	log.Printf("✅ Running on Node: %s (IP: %s)", nodeName, nodeIP)

	// 2. Configure In-Cluster Client for Kubernetes API
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("❌ ERROR: Failed to get in-cluster config: %v", err)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("❌ ERROR: Failed to create Kubernetes clientset: %v", err)
	}

	// --- 3. HTTP Server Phase ---
	http.HandleFunc("/", azHandler)
	log.Printf("🌍 Starting web server on port %s...", ListenPort)
	log.Fatal(http.ListenAndServe(":"+ListenPort, nil))
}

// azHandler handles HTTP requests and serves the full environment information.
func azHandler(w http.ResponseWriter, r *http.Request) {
	// Query the Kubernetes API for the Node object to get the AZ
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorMessage := fmt.Sprintf("❌ Error getting Node '%s'. Check RBAC permissions: %v", nodeName, err)
		log.Print(errorMessage)
		fmt.Fprintf(w, "<html><body><h1>500 Internal Server Error</h1><p>%s</p></body></html>", errorMessage)
		return
	}

	// Extract the Availability Zone from the Node's labels
	az, ok := node.Labels[AzLabelKey]
	if !ok {
		az = "AZ UNKNOWN (Label Not Found)"
	}

	// 5. Success - Build HTML Response
	w.WriteHeader(http.StatusOK)
	responseHtml := fmt.Sprintf(`<html><body>
		<h1>OpenShift Pod & Node Information</h1>
		
		<p><strong>Deployment Status:</strong> <span style="color: green;">Active</span></p>

		<h2>Pod Details</h2>
		<ul>
			<li><strong>Pod Name:</strong> %s</li>
			<li><strong>Namespace:</strong> %s</li>
			<li><strong>Pod IP:</strong> %s</li>
		</ul>
		
		<h2>Node Details</h2>
		<ul>
			<li><strong>Node Name:</strong> %s</li>
			<li><strong>Node IP:</strong> %s</li>
			<li><strong>Availability Zone (AZ):</strong> <strong style="color: blue;">%s</strong></li>
		</ul>
		
	</body></html>`, podName, podNamespace, podIP, nodeName, nodeIP, az)

	fmt.Fprint(w, responseHtml)
	log.Printf("Request served successfully. AZ: %s", az)
}