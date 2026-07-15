//go:build e2e

package webhook

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

var rayServiceGVR = schema.GroupVersionResource{
	Group:    "ray.io",
	Version:  "v1",
	Resource: "rayservices",
}

// waitForRayServiceReady is a helper function to poll and block until the RayService
// transitions into a Running or Ready status.
func waitForRayServiceReady(t *testing.T, serviceName string, timeout time.Duration) {
	t.Helper()
	t.Logf("Waiting for RayService %s to reach ready status...", serviceName)

	err := wait.PollUntilContextTimeout(t.Context(), 10*time.Second, timeout,
		true, func(ctx context.Context) (bool, error) {
			unstructSvc, err := dynamicClient.Resource(rayServiceGVR).
				Namespace(testNamespace).Get(ctx, serviceName, v1.GetOptions{})
			if err != nil {
				return false, err
			}

			status, found, err := unstructured.NestedString(unstructSvc.Object, "status", "serviceStatus")
			if err != nil || !found {
				t.Logf("RayService %s status not found yet.", serviceName)
				return false, nil
			}

			t.Logf("RayService %s current status: %s", serviceName, status)
			if status == "Running" || status == "Ready" { // Match what KubeRay produces
				return true, nil
			}
			if status == "Error" || status == "Failed" {
				return false, fmt.Errorf("RayService %s failed with status: %s", serviceName, status)
			}
			return false, nil
		})

	if err != nil {
		t.Fatalf("RayService %s did not become ready within %v: %v", serviceName, timeout, err)
	}
}

// waitForRayServiceHeadPod is a helper function to poll and retrieve the name of the
// running head Pod for a RayService.
func waitForRayServiceHeadPod(t *testing.T, serviceName string) string {
	t.Helper()
	var headPodName string
	err := wait.PollUntilContextTimeout(t.Context(), 1*time.Second, 120*time.Second,
		true, func(ctx context.Context) (bool, error) {
			pods, err := clientset.CoreV1().Pods(testNamespace).List(ctx, v1.ListOptions{
				LabelSelector: "ray.io/node-type=head",
			})
			if err != nil {
				return false, err
			}
			for _, pod := range pods.Items {
				if strings.HasPrefix(pod.Name, serviceName) && pod.Status.Phase == "Running" {
					headPodName = pod.Name
					return true, nil
				}
			}
			return false, nil
		})
	if err != nil {
		t.Fatalf("Failed to find Running head pod for RayService %s: %v", serviceName, err)
	}
	return headPodName
}

// verifyRayServiceCompletions is a helper function to send a test curl completion request
// to the vLLM server running on the head node and verify the response format.
func verifyRayServiceCompletions(t *testing.T, serviceName string) {
	t.Helper()
	// Fetch head pod targeting specifically this serviceName prefix
	headPodName := waitForRayServiceHeadPod(t, serviceName)

	// Curl the vLLM endpoint
	t.Logf("Sending curl request to %s...", headPodName)
	curlCmd := []string{
		"curl", "-s", "-X", "POST", "http://localhost:8000/v1/completions",
		"-H", "Content-Type: application/json",
		"-d", `{"model": "Qwen/Qwen1.5-0.5B-Chat", ` +
			`"prompt": "San Francisco is a", "max_tokens": 15, "temperature": 0}`,
	}
	stdout, stderr, err := execCommandInPod(t, headPodName, "ray-head", curlCmd)
	if err != nil {
		t.Fatalf("Failed to curl vLLM endpoint: %v\nStderr: %s", err, stderr)
	}
	t.Logf("vLLM response: %s", stdout)

	if !strings.Contains(stdout, "choices") {
		t.Fatalf("vLLM response did not contain 'choices'. Output: %s", stdout)
	}
}

func TestRayServiceIntegration_V6eSingleHost(t *testing.T) {
	serviceName := "v6e-rayservice-single"
	waitForRayServiceReady(t, serviceName, 60*time.Minute)
	verifyRayServiceCompletions(t, serviceName)
}

func TestRayServiceIntegration_V6eMultiHost(t *testing.T) {
	serviceName := "v6e-rayservice-multi"
	waitForRayServiceReady(t, serviceName, 60*time.Minute)
	verifyRayServiceCompletions(t, serviceName)
}
