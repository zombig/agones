// Copyright 2018 Google LLC All Rights Reserved.
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

package gameservers

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

var (
	n1 = corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1", UID: "node1"}}
	n2 = corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node2", UID: "node2"}}
	n3 = corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node3", UID: "node3"}}
)

func TestPortAllocatorAllocate(t *testing.T) {
	t.Parallel()
	fixture := dynamicGameServerFixture()

	t.Run("test allocated port counts", func(t *testing.T) {
		m := agtesting.NewMocks()
		pa := NewPortAllocator(10, 50, m.KubeInformerFactory, m.AgonesInformerFactory)
		nodeWatch := watch.NewFake()
		m.KubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))

		stop, cancel := agtesting.StartInformers(m, pa.nodeSynced)
		defer cancel()

		// Make sure the add's don't corrupt the sync
		// (no longer an issue, but leave this here for posterity)
		nodeWatch.Add(&n1)
		nodeWatch.Add(&n2)
		assert.True(t, cache.WaitForCacheSync(stop, pa.nodeSynced))

		err := pa.syncAll()
		assert.Nil(t, err)

		// single port dynamic
		pa.Allocate(fixture.DeepCopy())
		assert.Nil(t, err)
		assert.Equal(t, 1, countTotalAllocatedPorts(pa))

		pa.Allocate(fixture.DeepCopy())
		assert.Nil(t, err)
		assert.Equal(t, 2, countTotalAllocatedPorts(pa))

		// double port, dynamic
		gsCopy := fixture.DeepCopy()
		gsCopy.Spec.Ports = append(gsCopy.Spec.Ports, agonesv1.GameServerPort{Name: "another", ContainerPort: 6666, PortPolicy: agonesv1.Dynamic})
		assert.Len(t, gsCopy.Spec.Ports, 2)
		pa.Allocate(gsCopy.DeepCopy())
		assert.Nil(t, err)
		assert.Equal(t, 4, countTotalAllocatedPorts(pa))

		// three ports, dynamic
		gsCopy = gsCopy.DeepCopy()
		gsCopy.Spec.Ports = append(gsCopy.Spec.Ports, agonesv1.GameServerPort{Name: "another", ContainerPort: 6666, PortPolicy: agonesv1.Dynamic})
		assert.Len(t, gsCopy.Spec.Ports, 3)
		pa.Allocate(gsCopy)
		assert.Nil(t, err)
		assert.Equal(t, 7, countTotalAllocatedPorts(pa))

		// 4 ports, 1 static, rest dynamic
		gsCopy = gsCopy.DeepCopy()
		expected := int32(9999)
		gsCopy.Spec.Ports = append(gsCopy.Spec.Ports, agonesv1.GameServerPort{Name: "another", ContainerPort: 6666, HostPort: expected, PortPolicy: agonesv1.Static})
		assert.Len(t, gsCopy.Spec.Ports, 4)
		pa.Allocate(gsCopy)
		assert.Nil(t, err)
		assert.Equal(t, 10, countTotalAllocatedPorts(pa))
		assert.Equal(t, agonesv1.Static, gsCopy.Spec.Ports[3].PortPolicy)
		assert.Equal(t, expected, gsCopy.Spec.Ports[3].HostPort)

		// single port, passthrough
		gsCopy = fixture.DeepCopy()
		gsCopy.Spec.Ports[0] = agonesv1.GameServerPort{Name: "passthrough", PortPolicy: agonesv1.Passthrough}
		assert.Len(t, gsCopy.Spec.Ports, 1)
		pa.Allocate(gsCopy)
		assert.NotEmpty(t, gsCopy.Spec.Ports[0].HostPort)
		assert.Equal(t, gsCopy.Spec.Ports[0].HostPort, gsCopy.Spec.Ports[0].ContainerPort)
		assert.Nil(t, err)
		assert.Equal(t, 11, countTotalAllocatedPorts(pa))
	})

	t.Run("ports are all allocated", func(t *testing.T) {
		m := agtesting.NewMocks()
		pa := NewPortAllocator(10, 20, m.KubeInformerFactory, m.AgonesInformerFactory)
		nodeWatch := watch.NewFake()
		m.KubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))

		stop, cancel := agtesting.StartInformers(m, pa.nodeSynced)
		defer cancel()

		// Make sure the add's don't corrupt the sync
		nodeWatch.Add(&n1)
		nodeWatch.Add(&n2)
		assert.True(t, cache.WaitForCacheSync(stop, pa.nodeSynced))

		err := pa.syncAll()
		assert.Nil(t, err)

		// two nodes
		for x := 0; x < 2; x++ {
			// ports between 10 and 20
			for i := 10; i <= 20; i++ {
				var p int32
				gs := pa.Allocate(fixture.DeepCopy())
				assert.True(t, 10 <= gs.Spec.Ports[0].HostPort && gs.Spec.Ports[0].HostPort <= 20, "%v is not between 10 and 20", p)
				assert.Nil(t, err)
			}
		}

		assert.Len(t, pa.portAllocations, 2)
		gs := pa.Allocate(fixture.DeepCopy())
		assert.NotEmpty(t, gs.Spec.Ports[0].HostPort)
		assert.Len(t, pa.portAllocations, 3)
	})

	t.Run("ports are all allocated with multiple ports per GameServers", func(t *testing.T) {
		m := agtesting.NewMocks()
		maxPort := int32(19) // make sure we have an even number
		pa := NewPortAllocator(10, maxPort, m.KubeInformerFactory, m.AgonesInformerFactory)
		nodeWatch := watch.NewFake()
		m.KubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))

		stop, cancel := agtesting.StartInformers(m, pa.nodeSynced)
		defer cancel()

		morePortFixture := fixture.DeepCopy()
		morePortFixture.Spec.Ports = append(morePortFixture.Spec.Ports,
			agonesv1.GameServerPort{Name: "another", ContainerPort: 6666, PortPolicy: agonesv1.Dynamic},
			agonesv1.GameServerPort{Name: "static", ContainerPort: 6666, PortPolicy: agonesv1.Static, HostPort: 9999},
			agonesv1.GameServerPort{Name: "passthrough", PortPolicy: agonesv1.Passthrough})

		assert.Len(t, morePortFixture.Spec.Ports, 4)

		// Make sure the add's don't corrupt the sync
		nodeWatch.Add(&n1)
		nodeWatch.Add(&n2)
		assert.True(t, cache.WaitForCacheSync(stop, pa.nodeSynced))

		err := pa.syncAll()
		assert.Nil(t, err)

		// two nodes
		for x := 0; x < 2; x++ {
			// ports between 10 and 20, but there are 2 ports
			for i := 10; i <= 14; i++ {
				gsCopy := morePortFixture.DeepCopy()
				gsCopy.ObjectMeta.UID = types.UID(strconv.Itoa(x) + ":" + strconv.Itoa(i))
				gs := pa.Allocate(gsCopy)

				// Dynamic
				assert.NotEmpty(t, gs.Spec.Ports[0].HostPort)

				// Passthrough
				passThrough := gs.Spec.Ports[3]
				assert.Equal(t, agonesv1.Passthrough, passThrough.PortPolicy)
				assert.NotEmpty(t, passThrough.HostPort)
				assert.Equal(t, passThrough.HostPort, passThrough.ContainerPort)

				logrus.WithField("uid", gsCopy.ObjectMeta.UID).WithField("ports", gs.Spec.Ports).WithError(err).Info("Allocated Port")
				assert.Nil(t, err)
				for _, p := range gs.Spec.Ports {
					if p.PortPolicy == agonesv1.Dynamic || p.PortPolicy == agonesv1.Passthrough {
						assert.True(t, 10 <= p.HostPort && p.HostPort <= maxPort, "%v is not between 10 and 20", p)
					}
				}
			}
		}

		logrus.WithField("allocated", countTotalAllocatedPorts(pa)).WithField("count", len(pa.portAllocations[0])+len(pa.portAllocations[1])).Info("How many allocated")
		assert.Len(t, pa.portAllocations, 3)
		gs := pa.Allocate(fixture.DeepCopy())
		assert.NotEmpty(t, gs.Spec.Ports[0].HostPort)
		assert.Len(t, pa.portAllocations, 4)
	})

	t.Run("ports are unique in a node", func(t *testing.T) {
		fixture := dynamicGameServerFixture()
		m := agtesting.NewMocks()
		pa := NewPortAllocator(10, 20, m.KubeInformerFactory, m.AgonesInformerFactory)

		m.KubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
			nl := &corev1.NodeList{Items: []corev1.Node{n1}}
			return true, nl, nil
		})
		_, cancel := agtesting.StartInformers(m, pa.nodeSynced)
		defer cancel()
		err := pa.syncAll()
		assert.Nil(t, err)
		var ports []int32
		for i := 10; i <= 20; i++ {
			gs := pa.Allocate(fixture.DeepCopy())
			assert.Nil(t, err)
			assert.NotContains(t, ports, gs.Spec.Ports[0].HostPort)
			ports = append(ports, gs.Spec.Ports[0].HostPort)
		}
	})
}

func TestPortAllocatorMultithreadAllocate(t *testing.T) {
	fixture := dynamicGameServerFixture()
	m := agtesting.NewMocks()
	pa := NewPortAllocator(10, 20, m.KubeInformerFactory, m.AgonesInformerFactory)

	m.KubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: []corev1.Node{n1, n2}}
		return true, nl, nil
	})
	_, cancel := agtesting.StartInformers(m, pa.nodeSynced)
	defer cancel()
	err := pa.syncAll()
	assert.Nil(t, err)
	wg := sync.WaitGroup{}

	// do this for more than the nodes that are pre-allocated, to make sure
	// it works for dynamic node addition
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			for x := 0; x < 10; x++ {
				logrus.WithField("x", x).WithField("i", i).Info("allocating!")
				gs := pa.Allocate(fixture.DeepCopy())
				for _, p := range gs.Spec.Ports {
					assert.NotEmpty(t, p.HostPort)
				}
				assert.Nil(t, err)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestPortAllocatorDeAllocate(t *testing.T) {
	t.Parallel()

	fixture := dynamicGameServerFixture()
	m := agtesting.NewMocks()
	pa := NewPortAllocator(10, 20, m.KubeInformerFactory, m.AgonesInformerFactory)
	nodes := []corev1.Node{n1, n2, n3}
	m.KubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: nodes}
		return true, nl, nil
	})
	_, cancel := agtesting.StartInformers(m, pa.nodeSynced)
	defer cancel()
	err := pa.syncAll()
	assert.Nil(t, err)

	// gate
	assert.NotEmpty(t, fixture.Spec.Ports)

	for i := 0; i <= 100; i++ {
		gs := pa.Allocate(fixture.DeepCopy())
		assert.Nil(t, err)
		port := gs.Spec.Ports[0]
		assert.True(t, 10 <= port.HostPort && port.HostPort <= 20)
		assert.Equal(t, 1, countAllocatedPorts(pa, port.HostPort))
		assert.Len(t, pa.gameServerRegistry, 1)

		// test a non allocated
		nonAllocatedGS := gs.DeepCopy()
		nonAllocatedGS.ObjectMeta.Name = "no"
		nonAllocatedGS.ObjectMeta.UID = "no"
		pa.DeAllocate(nonAllocatedGS)
		assert.Equal(t, 1, countAllocatedPorts(pa, port.HostPort))
		assert.Len(t, pa.gameServerRegistry, 1)

		pa.DeAllocate(gs)
		assert.Equal(t, 0, countAllocatedPorts(pa, port.HostPort))
		assert.Len(t, pa.gameServerRegistry, 0)
	}
}

func TestPortAllocatorSyncPortAllocations(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	pa := NewPortAllocator(10, 20, m.KubeInformerFactory, m.AgonesInformerFactory)

	m.KubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: []corev1.Node{n1, n2, n3}}
		return true, nl, nil
	})

	m.AgonesClient.AddReactor("list", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs1 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", UID: "1"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 10}},
			},
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 10}}, NodeName: n1.ObjectMeta.Name}}
		gs2 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", UID: "2"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 10}},
			},
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 10}}, NodeName: n2.ObjectMeta.Name}}
		gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3", UID: "3"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 11}},
			},
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 11}}, NodeName: n3.ObjectMeta.Name}}
		gs4 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4", UID: "4"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Passthrough, HostPort: 12}},
			},
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
		gs5 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs5", UID: "5"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 12}},
			},
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
		gs6 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs6", UID: "6"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Static, HostPort: 12}},
			},
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
		gsl := &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs1, gs2, gs3, gs4, gs5, gs6}}
		return true, gsl, nil
	})

	_, cancel := agtesting.StartInformers(m, pa.gameServerSynced, pa.nodeSynced)
	defer cancel()

	err := pa.syncAll()
	assert.Nil(t, err)

	assert.Len(t, pa.portAllocations, 3)
	assert.Len(t, pa.gameServerRegistry, 5)

	// count the number of allocated ports,
	assert.Equal(t, 2, countAllocatedPorts(pa, 10))
	assert.Equal(t, 1, countAllocatedPorts(pa, 11))
	assert.Equal(t, 2, countAllocatedPorts(pa, 12))

	count := 0
	for i := int32(10); i <= 20; i++ {
		count += countAllocatedPorts(pa, i)
	}
	assert.Equal(t, 5, count)
}

func TestPortAllocatorSyncDeleteGameServer(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	gs1 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", UID: "1"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 10}},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 10}}, NodeName: n1.ObjectMeta.Name}}
	gs2 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", UID: "2"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 11}},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 11}}, NodeName: n1.ObjectMeta.Name}}
	gs3 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3", UID: "3"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Passthrough, HostPort: 10}},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 10}}, NodeName: n2.ObjectMeta.Name}}
	gs4 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4", UID: "4"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 10}},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 10}}, NodeName: n2.ObjectMeta.Name}}

	pa := NewPortAllocator(10, 20, m.KubeInformerFactory, m.AgonesInformerFactory)

	m.KubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: []corev1.Node{n1, n2, n3}}
		return true, nl, nil
	})

	stop, cancel := agtesting.StartInformers(m, pa.gameServerSynced, pa.nodeSynced)
	defer cancel()

	gsWatch.Add(gs1.DeepCopy())
	gsWatch.Add(gs2.DeepCopy())
	gsWatch.Add(gs3.DeepCopy())

	assert.True(t, cache.WaitForCacheSync(stop, pa.gameServerSynced))

	err := pa.syncAll()
	assert.Nil(t, err)

	// gate
	pa.mutex.RLock() // reading mutable state, so read lock
	assert.Equal(t, 2, countAllocatedPorts(pa, 10))
	assert.Equal(t, 1, countAllocatedPorts(pa, 11))
	pa.mutex.RUnlock()

	// delete allocated gs
	gsWatch.Delete(gs3.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(stop, pa.gameServerSynced))

	pa.mutex.RLock() // reading mutable state, so read lock
	assert.Equal(t, 1, countAllocatedPorts(pa, 10))
	assert.Equal(t, 1, countAllocatedPorts(pa, 11))
	pa.mutex.RUnlock()

	// delete the currently non allocated server, all should be the same
	// simulated getting an old delete message
	gsWatch.Delete(gs4.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(stop, pa.gameServerSynced))
	pa.mutex.RLock() // reading mutable state, so read lock
	assert.Equal(t, 1, countAllocatedPorts(pa, 10))
	assert.Equal(t, 1, countAllocatedPorts(pa, 11))
	pa.mutex.RUnlock()
}

func TestNodePortAllocation(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	pa := NewPortAllocator(10, 20, m.KubeInformerFactory, m.AgonesInformerFactory)
	nodes := []corev1.Node{n1, n2, n3}
	m.KubeClient.AddReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		nl := &corev1.NodeList{Items: nodes}
		return true, nl, nil
	})
	result := pa.nodePortAllocation([]*corev1.Node{&n1, &n2, &n3})
	assert.Len(t, result, 3)
	for _, n := range nodes {
		ports, ok := result[n.ObjectMeta.Name]
		assert.True(t, ok, "Should have a port allocation for %s", n.ObjectMeta.Name)
		assert.Len(t, ports, 11)
		for _, v := range ports {
			assert.False(t, v)
		}
	}
}

func TestTakePortAllocation(t *testing.T) {
	t.Parallel()

	fixture := []portAllocation{{1: false, 2: false}, {1: false, 2: false}, {1: false, 3: false}}
	result := setPortAllocation(2, fixture, true)
	assert.True(t, result[0][2])

	for i, row := range fixture {
		for p, taken := range row {
			if i != 0 && p != 2 {
				assert.False(t, taken, fmt.Sprintf("row %d and port %d should be false", i, p))
			}
		}
	}
}

func TestPortAllocatorRegisterExistingGameServerPorts(t *testing.T) {
	t.Parallel()
	m := agtesting.NewMocks()
	pa := NewPortAllocator(10, 13, m.KubeInformerFactory, m.AgonesInformerFactory)

	gs1 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", UID: "1"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 10}},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 10}}, NodeName: n1.ObjectMeta.Name}}

	gs2 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", UID: "2"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 11}},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 11}}, NodeName: n2.ObjectMeta.Name}}

	gs3 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3", UID: "3"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Passthrough, HostPort: 12}},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, Ports: []agonesv1.GameServerStatusPort{{Port: 12}}, NodeName: n1.ObjectMeta.Name}}

	gs4 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4", UID: "3"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, HostPort: 13}},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStatePortAllocation, Ports: []agonesv1.GameServerStatusPort{{Port: 13}}}}

	allocations, nonReadyNodesPorts := pa.registerExistingGameServerPorts([]*agonesv1.GameServer{gs1, gs2, gs3, gs4}, []*corev1.Node{&n1, &n2, &n3}, map[types.UID]bool{})

	assert.Equal(t, []int32{13}, nonReadyNodesPorts)
	assert.Equal(t, portAllocation{10: true, 11: false, 12: true, 13: false}, allocations[0])
	assert.Equal(t, portAllocation{10: false, 11: true, 12: false, 13: false}, allocations[1])
	assert.Equal(t, portAllocation{10: false, 11: false, 12: false, 13: false}, allocations[2])
}

func dynamicGameServerFixture() *agonesv1.GameServer {
	return &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "1234"},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Dynamic, ContainerPort: 7777}},
		},
	}
}

// countAllocatedPorts counts how many of a given port have been
// allocated across nodes
func countAllocatedPorts(pa *PortAllocator, p int32) int {
	count := 0
	for _, node := range pa.portAllocations {
		if node[p] {
			count++
		}
	}
	return count
}

// countTotalAllocatedPorts counts the total number of allocated ports
func countTotalAllocatedPorts(pa *PortAllocator) int {
	count := 0
	for _, node := range pa.portAllocations {
		for _, alloc := range node {
			if alloc {
				count++
			}
		}
	}
	return count
}
