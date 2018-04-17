// Copyright 2018 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This code was autogenerated. Do not edit directly.
package fake

import (
	v1alpha1 "agones.dev/agones/pkg/apis/stable/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGameServerSets implements GameServerSetInterface
type FakeGameServerSets struct {
	Fake *FakeStableV1alpha1
	ns   string
}

var gameserversetsResource = schema.GroupVersionResource{Group: "stable.agones.dev", Version: "v1alpha1", Resource: "gameserversets"}

var gameserversetsKind = schema.GroupVersionKind{Group: "stable.agones.dev", Version: "v1alpha1", Kind: "GameServerSet"}

// Get takes name of the gameServerSet, and returns the corresponding gameServerSet object, and an error if there is any.
func (c *FakeGameServerSets) Get(name string, options v1.GetOptions) (result *v1alpha1.GameServerSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(gameserversetsResource, c.ns, name), &v1alpha1.GameServerSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GameServerSet), err
}

// List takes label and field selectors, and returns the list of GameServerSets that match those selectors.
func (c *FakeGameServerSets) List(opts v1.ListOptions) (result *v1alpha1.GameServerSetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(gameserversetsResource, gameserversetsKind, c.ns, opts), &v1alpha1.GameServerSetList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.GameServerSetList{}
	for _, item := range obj.(*v1alpha1.GameServerSetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested gameServerSets.
func (c *FakeGameServerSets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(gameserversetsResource, c.ns, opts))

}

// Create takes the representation of a gameServerSet and creates it.  Returns the server's representation of the gameServerSet, and an error, if there is any.
func (c *FakeGameServerSets) Create(gameServerSet *v1alpha1.GameServerSet) (result *v1alpha1.GameServerSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(gameserversetsResource, c.ns, gameServerSet), &v1alpha1.GameServerSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GameServerSet), err
}

// Update takes the representation of a gameServerSet and updates it. Returns the server's representation of the gameServerSet, and an error, if there is any.
func (c *FakeGameServerSets) Update(gameServerSet *v1alpha1.GameServerSet) (result *v1alpha1.GameServerSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(gameserversetsResource, c.ns, gameServerSet), &v1alpha1.GameServerSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GameServerSet), err
}

// Delete takes name of the gameServerSet and deletes it. Returns an error if one occurs.
func (c *FakeGameServerSets) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(gameserversetsResource, c.ns, name), &v1alpha1.GameServerSet{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGameServerSets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(gameserversetsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.GameServerSetList{})
	return err
}

// Patch applies the patch and returns the patched gameServerSet.
func (c *FakeGameServerSets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.GameServerSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(gameserversetsResource, c.ns, name, data, subresources...), &v1alpha1.GameServerSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.GameServerSet), err
}