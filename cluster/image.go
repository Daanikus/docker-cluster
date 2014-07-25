// Copyright 2014 docker-cluster authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cluster

import (
	"github.com/fsouza/go-dockerclient"
)

// RemoveImage removes an image from the nodes where this images exists, returning an
// error in case of failure. Will wait for the image to be removed from all nodes.
func (c *Cluster) RemoveImageWait(name string) error {
	return c.removeImage(name, true)
}

// RemoveImage removes an image from the nodes where this images exists, returning an
// error in case of failure. Will wait for the image to be removed only from one node,
// removal from the other nodes will happen in background.
func (c *Cluster) RemoveImage(name string) error {
	return c.removeImage(name, false)
}

func (c *Cluster) removeImage(name string, waitForAll bool) error {
	hosts, err := c.storage().RetrieveImage(name)
	if err != nil {
		return err
	}
	_, err = c.runOnNodes(func(n node) (interface{}, error) {
		return nil, n.RemoveImage(name)
	}, docker.ErrNoSuchImage, waitForAll, hosts...)
	if err == nil || err == docker.ErrNoSuchImage {
		otherErr := c.storage().RemoveImage(name)
		if otherErr != nil {
			return otherErr
		}
	}
	return err
}

// PullImage pulls an image from a remote registry server, returning an error
// in case of failure.
//
// It will pull all images in parallel, so users need to make sure that the
// given buffer is safe.
func (c *Cluster) PullImage(opts docker.PullImageOptions, auth docker.AuthConfiguration, nodes ...string) error {
	_, err := c.runOnNodes(func(n node) (interface{}, error) {
		key := opts.Repository
		c.storage().StoreImage(key, n.addr)
		return nil, n.PullImage(opts, auth)
	}, docker.ErrNoSuchImage, true, nodes...)
	return err
}

// PushImage pushes an image to a remote registry server, returning an error in
// case of failure.
func (c *Cluster) PushImage(opts docker.PushImageOptions, auth docker.AuthConfiguration) error {
	nodes, err := c.getNodesForImage(opts.Name)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		return node.PushImage(opts, auth)
	}
	return nil
}

func (c *Cluster) getNodesForImage(image string) ([]node, error) {
	var nodes []node
	hosts, err := c.storage().RetrieveImage(image)
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		node, err := c.getNode(func(s Storage) (string, error) { return host, nil })
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, err
}

// ImportImage imports an image from a url or stdin
func (c *Cluster) ImportImage(opts docker.ImportImageOptions) error {
	_, err := c.runOnNodes(func(n node) (interface{}, error) {
		return nil, n.ImportImage(opts)
	}, docker.ErrNoSuchImage, false)
	return err
}

//BuildImage build an image and push it to register
func (c *Cluster) BuildImage(buildOptions docker.BuildImageOptions) error {
	nodes, err := c.Nodes()
	if err != nil {
		return err
	}
	nodeAddress := nodes[0].Address
	node, err := c.getNode(func(Storage) (string, error) {
		return nodeAddress, nil
	})
	if err != nil {
		return err
	}
	err = node.BuildImage(buildOptions)
	if err != nil {
		return err
	}
	return c.storage().StoreImage(buildOptions.Name, nodeAddress)
}
