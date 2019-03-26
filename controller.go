package main

import (
	"errors"
	"io"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

type controller struct {
	cs          kubernetes.Interface
	nodesLister corev1listers.NodeLister
	nodesSynced cache.InformerSynced
	hosts       hostsFile
	hostsWriter io.Writer
}

func newController(cs kubernetes.Interface, nodesInformer coreinformers.NodeInformer, hostsWriter io.Writer) *controller {
	con := &controller{
		cs:          cs,
		nodesLister: nodesInformer.Lister(),
		nodesSynced: nodesInformer.Informer().HasSynced,
		hosts:       hostsFile{},
		hostsWriter: hostsWriter,
	}

	klog.Info("Setting up event handlers")
	nodesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: con.upsertNode,
		UpdateFunc: func(old, new interface{}) {
			newNode := new.(*corev1.Node)
			oldNode := old.(*corev1.Node)
			if newNode.ResourceVersion == oldNode.ResourceVersion {
				// Periodic resync will send update events for all known Nodes.
				// Two different versions of the same Node will always have different RVs.
				return
			}
			con.upsertNode(new)
		},
		DeleteFunc: con.removeNode,
	})

	return con
}

// Run starts the controller loop until stopCh is closed.
// When non-nil, syncDoneCh will be closed as soon as cache synchronization
// is complete. (This is helpful for testing purposes.)
func (c *controller) Run(stopCh <-chan struct{}, syncDoneCh chan struct{}) error {
	defer runtime.HandleCrash()

	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.nodesSynced); !ok {
		return errors.New("Informer caches did not sync in time")
	}
	klog.Info("Informer caches synced successfully")
	if syncDoneCh != nil {
		close(syncDoneCh)
	}

	<-stopCh
	klog.Info("Shutting down controller")

	return nil
}

func (c *controller) upsertNode(obj interface{}) {
	node := obj.(*corev1.Node)
	rec := toHostRecord(node)
	c.updateHostsFileIfNeeded(&rec)
}

func (c *controller) removeNode(obj interface{}) {
	var rec hostRecord
	unknown, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		rec.hostname = unknown.Key
	} else {
		node := obj.(*corev1.Node)
		rec.hostname = node.ObjectMeta.Name
	}
	c.updateHostsFileIfNeeded(&rec)
}

// updateHostsFileIfNeeded updates the hosts file if the new version is
// different from the current one. It does so by creating a copy from
// the current version, applies the change, compares the result to the
// current version, and writes out the new version if a change has been
// observed.
// The method handles both hosts file additions and removals. The latter are
// characterized by a host record with an empty IP address.
func (c *controller) updateHostsFileIfNeeded(rec *hostRecord) {
	newHosts := make(hostsFile, len(c.hosts))
	for k, v := range c.hosts {
		newHosts[k] = v
	}

	if rec.ipAddr == "" {
		delete(newHosts, rec.hostname)
	} else {
		newHosts[rec.hostname] = rec.ipAddr
	}

	if reflect.DeepEqual(newHosts, c.hosts) {
		klog.V(3).Info("hosts file did not change -- skipping update.")
		return
	}
	c.hosts = newHosts

	klog.Infof("Updating hosts file with: [%s]", rec)
	if _, err := c.hostsWriter.Write([]byte(c.hosts.String())); err != nil {
		klog.Errorf("Failed to write hosts file: %s", err)
	}
}
