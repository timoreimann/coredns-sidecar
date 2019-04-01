package main

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type testWriter struct {
	rendered chan string
	stopOn   int
}

func newTestWriter(stopOn int) *testWriter {
	if stopOn <= 0 {
		panic("stopOn must be >= 0")
	}

	return &testWriter{
		rendered: make(chan string, 1),
		stopOn:   stopOn,
	}
}

func (tw *testWriter) Write(p []byte) (int, error) {
	tw.rendered <- string(p)
	tw.stopOn--
	if tw.stopOn == 0 {
		close(tw.rendered)
	}
	return len(p), nil
}

func createInternalNode(name string, addr string) *corev1.Node {
	return createNode(name, []corev1.NodeAddress{
		{
			Type:    corev1.NodeInternalIP,
			Address: addr,
		},
	})
}

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		nodeMutator func(cs kubernetes.Interface) error
		stopOn      int
		wantHosts   string
	}{
		{
			name: "add nodes",
			nodeMutator: func(cs kubernetes.Interface) error {
				node1 := createInternalNode("host1", "1.1.1.1")
				node2 := createInternalNode("host2", "2.2.2.2")

				for i, node := range []*corev1.Node{node1, node2} {
					i++
					if _, err := cs.CoreV1().Nodes().Create(node); err != nil {
						return fmt.Errorf("failed to create node #%d: %s", i, err)
					}
				}
				return nil
			},
			stopOn: 2,
			wantHosts: hostsFile{
				"host1": "1.1.1.1",
				"host2": "2.2.2.2",
			}.String(),
		},
		{
			name: "delete node",
			nodeMutator: func(cs kubernetes.Interface) error {
				node1 := createInternalNode("host1", "1.1.1.1")
				node2 := createInternalNode("host2", "2.2.2.2")

				for i, node := range []*corev1.Node{node1, node2} {
					i++
					if _, err := cs.CoreV1().Nodes().Create(node); err != nil {
						return fmt.Errorf("failed to create node #%d: %s", i, err)
					}
				}

				err := cs.CoreV1().Nodes().Delete("host2", &metav1.DeleteOptions{})
				if err != nil {
					return fmt.Errorf("failed to delete node2: %s", err)
				}
				return nil
			},
			stopOn: 3,
			wantHosts: hostsFile{
				"host1": "1.1.1.1",
			}.String(),
		},
		{
			name: "no update",
			nodeMutator: func(cs kubernetes.Interface) error {
				node1 := createInternalNode("host1", "1.1.1.1")
				node2 := createInternalNode("host2", "2.2.2.2")

				_, err := cs.CoreV1().Nodes().Create(node1)
				if err != nil {
					return fmt.Errorf("failed to create node1: %s", err)
				}

				_, err = cs.CoreV1().Nodes().Update(node1)
				if err != nil {
					return fmt.Errorf("failed to update node1: %s", err)
				}

				node1.ResourceVersion = "42"
				_, err = cs.CoreV1().Nodes().Update(node1)
				if err != nil {
					return fmt.Errorf("failed to update node1 with different resource version: %s", err)
				}

				_, err = cs.CoreV1().Nodes().Create(node2)
				if err != nil {
					return fmt.Errorf("failed to create node2: %s", err)
				}
				return nil
			},
			stopOn: 2,
			wantHosts: hostsFile{
				"host1": "1.1.1.1",
				"host2": "2.2.2.2",
			}.String(),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cs := fake.NewSimpleClientset()
			informerFactory := informers.NewSharedInformerFactory(cs, 0)
			writer := newTestWriter(test.stopOn)
			stopCh := make(chan struct{}, 1)

			con := newController(cs, informerFactory.Core().V1().Nodes(), writer)
			informerFactory.Start(stopCh)

			syncDoneCh := make(chan struct{}, 1)
			go func() {
				con.Run(stopCh, syncDoneCh)
			}()
			defer func() {
				close(stopCh)
			}()

			// Wait for caches to sync so that no injected objects are skipped.
			<-syncDoneCh

			if err := test.nodeMutator(cs); err != nil {
				t.Fatal(err)
			}

			timeout := time.After(3 * time.Second)
			var gotHosts string
		Loop:
			for {
				select {
				case hosts, ok := <-writer.rendered:
					if !ok {
						break Loop
					}
					gotHosts = hosts
				case <-timeout:
					t.Fatal("timeout fired")
				}
			}

			if gotHosts != test.wantHosts {
				t.Error(hostsFileDiff(gotHosts, test.wantHosts))
			}
		})
	}
}
