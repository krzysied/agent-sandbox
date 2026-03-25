// Copyright 2026 The Kubernetes Authors.
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

package queue

import (
	"fmt"
	"sync"

	"k8s.io/client-go/util/workqueue"
)

type SimpleSandboxQueue struct {
	mu     sync.Mutex
	queues map[string]*synchronizedQueue
}

type synchronizedQueue struct {
	getMu sync.Mutex
	queue workqueue.TypedInterface[SandboxKey]
}

func NewSimplePodQueue() *SimpleSandboxQueue {
	return &SimpleSandboxQueue{
		queues: map[string]*synchronizedQueue{},
	}
}

func (s *SimpleSandboxQueue) Add(templateHash string, pod SandboxKey) {
	sQueue := s.getQueue(templateHash)
	sQueue.queue.Add(pod)
}

func (s *SimpleSandboxQueue) Get(templateHash string) (SandboxKey, bool) {
	sQueue := s.getQueue(templateHash)
	sQueue.getMu.Lock()
	defer sQueue.getMu.Unlock()
	if sQueue.queue.Len() == 0 {
		return SandboxKey{}, false
	}
	pod, shutdown := sQueue.queue.Get()
	if shutdown {
		return SandboxKey{}, false
	}
	return pod, true
}

func (s *SimpleSandboxQueue) Done(templateHash string, pod SandboxKey) {
	sQueue := s.getQueue(templateHash)
	sQueue.queue.Done(pod)
}

func (s *SimpleSandboxQueue) Len(templateHash string) int {
	sQueue := s.getQueue(templateHash)
	return sQueue.queue.Len()
}

func (s *SimpleSandboxQueue) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sQueue := range s.queues {
		sQueue.queue.ShutDown()
	}
}

func (s *SimpleSandboxQueue) getQueue(templateHash string) *synchronizedQueue {
	s.mu.Lock()
	defer s.mu.Unlock()
	sQueue, ok := s.queues[templateHash]
	if !ok {
		config := workqueue.TypedQueueConfig[SandboxKey]{
			Name: fmt.Sprintf("Pod Queue: %v", templateHash),
		}
		sQueue = &synchronizedQueue{
			queue: workqueue.NewTypedWithConfig(config),
		}
		s.queues[templateHash] = sQueue
	}
	return sQueue
}
