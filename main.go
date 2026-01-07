package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Define pod and container details
	podName := "logging-mock-service-pod"
	containerName := "logging-mock-service" // Optional: specify if multiple containers exist
	namespace := "default"

	// Set up Kubernetes configuration
	var config *rest.Config
	var err error

	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	// Check if kubeconfig file exists, otherwise use in-cluster config
	if _, err := os.Stat(kubeconfig); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	// Create the Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}

	// Set up log options with the 'Follow' flag for tailing
	podLogOptions := corev1.PodLogOptions{
		Container: containerName,
		Follow:    true,          // This is key for 'tailing' or streaming logs in real time
		TailLines: int64Ptr(100), // Optional: start with the last 100 lines
	}

	// Get the log stream request
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)

	// Use a context to allow cancellation if needed
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Retrieve the stream
	podLogs, err := req.Stream(ctx)
	if err != nil {
		log.Fatalf("Error in opening stream: %v", err)
	}
	defer podLogs.Close()

	scanner := bufio.NewScanner(podLogs)

	levelCounts := map[string]int64{
		"fatal": 0,
		"error": 0,
		"warn":  0,
		"info":  0,
		"debug": 0,
		"trace": 0,
	}

	var lineCount int64
	for scanner.Scan() {

		logLine := scanner.Text()
		logParts := strings.Split(logLine, " ")
		time, levelGroup, msgParts := logParts[0], logParts[1], logParts[2:]
		msg := strings.Join(msgParts, " ")

		levelParts := strings.Split(levelGroup, "=")

		// TODO: make pretty by putting this in a channel that would update
		// a TUI sub-window with a histogram and count with nice colors
		// Output periodically to STDOUT for now
		levelCounts[levelParts[1]]++
		if lineCount%60 == 0 {
			fmt.Printf(
				"TOTALS:\n fatal %8d\n error %8d\n warn  %8d\n info  %8d\n debug %8d\n trace %8d\n",
				levelCounts["fatal"],
				levelCounts["error"],
				levelCounts["warn"],
				levelCounts["info"],
				levelCounts["debug"],
				levelCounts["trace"],
			)
		}

		fmt.Printf("%s -- %s -- %s\n", time, levelGroup, msg)
		lineCount++
	}
}

// Helper function to return a pointer to an int64 for optional fields
func int64Ptr(i int64) *int64 {
	return &i
}
