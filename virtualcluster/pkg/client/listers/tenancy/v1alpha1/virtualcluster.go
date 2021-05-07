/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/apis/tenancy/v1alpha1"
)

// VirtualClusterLister helps list VirtualClusters.
type VirtualClusterLister interface {
	// List lists all VirtualClusters in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.VirtualCluster, err error)
	// VirtualClusters returns an object that can list and get VirtualClusters.
	VirtualClusters(namespace string) VirtualClusterNamespaceLister
	VirtualClusterListerExpansion
}

// virtualClusterLister implements the VirtualClusterLister interface.
type virtualClusterLister struct {
	indexer cache.Indexer
}

// NewVirtualClusterLister returns a new VirtualClusterLister.
func NewVirtualClusterLister(indexer cache.Indexer) VirtualClusterLister {
	return &virtualClusterLister{indexer: indexer}
}

// List lists all VirtualClusters in the indexer.
func (s *virtualClusterLister) List(selector labels.Selector) (ret []*v1alpha1.VirtualCluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.VirtualCluster))
	})
	return ret, err
}

// VirtualClusters returns an object that can list and get VirtualClusters.
func (s *virtualClusterLister) VirtualClusters(namespace string) VirtualClusterNamespaceLister {
	return virtualClusterNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// VirtualClusterNamespaceLister helps list and get VirtualClusters.
type VirtualClusterNamespaceLister interface {
	// List lists all VirtualClusters in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.VirtualCluster, err error)
	// Get retrieves the VirtualCluster from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.VirtualCluster, error)
	VirtualClusterNamespaceListerExpansion
}

// virtualClusterNamespaceLister implements the VirtualClusterNamespaceLister
// interface.
type virtualClusterNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all VirtualClusters in the indexer for a given namespace.
func (s virtualClusterNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.VirtualCluster, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.VirtualCluster))
	})
	return ret, err
}

// Get retrieves the VirtualCluster from the indexer for a given namespace and name.
func (s virtualClusterNamespaceLister) Get(name string) (*v1alpha1.VirtualCluster, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("virtualcluster"), name)
	}
	return obj.(*v1alpha1.VirtualCluster), nil
}
