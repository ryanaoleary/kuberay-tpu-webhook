//go:build e2e

package webhook

import (
	"testing"

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	errTopologyMismatch = "Number of workers in worker group not equal to specified topology"
)

// assertRayClusterRejected is a helper function to verify that creating a RayCluster
// with invalid configurations is rejected by the validating webhook.
func assertRayClusterRejected(t *testing.T, cluster *rayv1.RayCluster, expectedErrorMessage string) {
	t.Helper()
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
	if err != nil {
		t.Fatalf("Error converting RayCluster to unstructured: %v", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "ray.io",
		Version:  "v1",
		Resource: "rayclusters",
	}

	res, err := dynamicClient.Resource(gvr).Namespace(testNamespace).
		Create(t.Context(), &unstructured.Unstructured{Object: unstructuredObj}, metav1.CreateOptions{})
	if err == nil {
		t.Logf("Cleaning up leaked RayCluster %q...", res.GetName())
		_ = dynamicClient.Resource(gvr).Namespace(testNamespace).
			Delete(t.Context(), res.GetName(), metav1.DeleteOptions{})
		t.Fatalf("Expected cluster creation to fail due to webhook validation, but it succeeded")
	}

	statusErr, ok := err.(*apierrors.StatusError)
	if !ok {
		t.Fatalf("Expected StatusError from API server, got: %T (error: %v)", err, err)
	}

	assert.Contains(
		t,
		statusErr.Error(),
		expectedErrorMessage,
		"Validation rejected, but with unexpected error message",
	)
}

func TestRayClusterValidation_InvalidTopology(t *testing.T) {
	baseCluster := loadManifest(t, "../manifests/invalid/invalid-topology.yaml")

	t.Log("Running validating webhook case: Strict Topology Mismatch Rejection")
	// Base invalid-topology manifest has replica workerGroup with numOfHosts: 2
	// but requests 2x4 topology (expected 1 host)
	assertRayClusterRejected(t, baseCluster, errTopologyMismatch)
}

func TestRayClusterValidation_MissingTopologyKey(t *testing.T) {
	cluster := loadManifest(t, "../manifests/invalid/invalid-topology.yaml")
	cluster.Name = "tpu-v6e-missing-topology"

	t.Log("Running validating webhook case: Missing cloud.google.com/gke-tpu-topology Selector Rejection")
	// Remove topology nodeSelector key entirely
	delete(cluster.Spec.WorkerGroupSpecs[0].Template.Spec.NodeSelector, "cloud.google.com/gke-tpu-topology")

	// Since missing node selectors trigger an internal parsing error in checkWorkersMatchTopology,
	// the webhook controller terminates with an admission crash.
	// We assert that the API server returns a clean calling webhook crash payload.
	assertRayClusterRejected(t, cluster, "Failed to validate RayCluster")
}

func TestRayClusterValidation_NumOfHostsZero(t *testing.T) {
	cluster := loadManifest(t, "../manifests/invalid/invalid-num-of-hosts-zero.yaml")
	t.Log("Running validating webhook case: NumOfHosts=0 Rejection")
	assertRayClusterRejected(t, cluster, errTopologyMismatch)
}

func TestRayClusterValidation_NumOfHostsOmitted(t *testing.T) {
	cluster := loadManifest(t, "../manifests/invalid/invalid-num-of-hosts-omitted.yaml")
	t.Log("Running validating webhook case: NumOfHosts omitted Rejection")
	assertRayClusterRejected(t, cluster, errTopologyMismatch)
}
