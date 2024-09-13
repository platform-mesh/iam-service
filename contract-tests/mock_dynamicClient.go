package contract_tests

import (
	"context"
	"encoding/base64"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

type mockDynamicClient struct {
	dynamic.Interface
	fake *dynamicfake.FakeDynamicClient
}

func (m *mockDynamicClient) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return &mockNamespaceableResourceClient{
		NamespaceableResourceInterface: m.fake.Resource(resource),
		resource:                       resource,
	}
}

type mockNamespaceableResourceClient struct {
	dynamic.NamespaceableResourceInterface
	resource schema.GroupVersionResource
}

func (m *mockNamespaceableResourceClient) Namespace(ns string) dynamic.ResourceInterface {
	return &mockResourceClient{
		ResourceInterface: m.NamespaceableResourceInterface.Namespace(ns),
		resource:          m.resource,
		namespace:         ns,
	}
}

type mockResourceClient struct {
	dynamic.ResourceInterface
	resource  schema.GroupVersionResource
	namespace string
}

func (m *mockResourceClient) Get(ctx context.Context, name string, options v1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	switch m.resource.Resource {
	case "serviceaccounts":
		return &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ServiceAccount",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": m.namespace,
				},
				"secrets": []interface{}{
					map[string]interface{}{
						"name": "sa-secret",
					},
				},
			},
		}, nil
	case "secrets":
		return &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": m.namespace,
				},
				"data": map[string]interface{}{
					"ca.crt": base64.StdEncoding.EncodeToString([]byte("mock-ca-data")),
					"token":  base64.StdEncoding.EncodeToString([]byte("mock-token-data")),
				},
			},
		}, nil
	case "configmaps":
		if name == "shoot-info" && m.namespace == "kube-system" {
			return &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":      name,
						"namespace": m.namespace,
					},
					"data": map[string]interface{}{
						"domain":      "example.com",
						"shootName":   "mock-shoot",
						"projectName": "mock-project",
					},
				},
			}, nil
		}
	case "accounts":
		return &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Account",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": m.namespace,
				},
				"spec": map[string]interface{}{
					"namespace": "mock-namespace",
				},
			},
		}, nil
	}
	return nil, fmt.Errorf("resource not found")
}

func (m *mockResourceClient) List(ctx context.Context, opts v1.ListOptions) (*unstructured.UnstructuredList, error) {
	// Mock implementation for List
	return &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ServiceAccount",
					"metadata": map[string]interface{}{
						"name":      "sa-1",
						"namespace": "default",
						"labels": map[string]interface{}{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
			},
			{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ServiceAccount",
					"metadata": map[string]interface{}{
						"name":      "sa-2",
						"namespace": "default",
						"labels": map[string]interface{}{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
			},
		},
	}, nil
}

func createMockDynamicClient() dynamic.Interface {
	scheme := runtime.NewScheme()
	return &mockDynamicClient{
		fake: dynamicfake.NewSimpleDynamicClient(scheme),
	}
}
