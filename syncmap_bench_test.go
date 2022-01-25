// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// SPDX-FileCopyrightText: 2009 The Go Authors
//
// SPDX-License-Identifier: BSD-3-Clause

package typ_test

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"gopkg.in/typ.v1"
)

type bench[K comparable, V any] struct {
	setup func(*testing.B, mapInterface[K, V])
	perG  func(b *testing.B, pb *testing.PB, i int, m mapInterface[K, V])
}

func benchMap[K comparable, V any](b *testing.B, bench bench[K, V]) {
	for _, m := range [...]mapInterface[K, V]{&DeepCopyMap[K, V]{}, &RWMutexMap[K, V]{}, &typ.SyncMap[K, V]{}} {
		b.Run(fmt.Sprintf("%T", m), func(b *testing.B) {
			m = reflect.New(reflect.TypeOf(m).Elem()).Interface().(mapInterface[K, V])
			if bench.setup != nil {
				bench.setup(b, m)
			}

			b.ResetTimer()

			var i int64
			b.RunParallel(func(pb *testing.PB) {
				id := int(atomic.AddInt64(&i, 1) - 1)
				bench.perG(b, pb, id*b.N, m)
			})
		})
	}
}

func BenchmarkLoadMostlyHits(b *testing.B) {
	const hits, misses = 1023, 1

	benchMap(b, bench[int, int]{
		setup: func(_ *testing.B, m mapInterface[int, int]) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.Load(i % (hits + misses))
			}
		},
	})
}

func BenchmarkLoadMostlyMisses(b *testing.B) {
	const hits, misses = 1, 1023

	benchMap(b, bench[int, int]{
		setup: func(_ *testing.B, m mapInterface[int, int]) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.Load(i % (hits + misses))
			}
		},
	})
}

func BenchmarkLoadOrStoreBalanced(b *testing.B) {
	const hits, misses = 128, 128

	benchMap(b, bench[int, int]{
		setup: func(b *testing.B, m mapInterface[int, int]) {
			if _, ok := m.(*DeepCopyMap[int, int]); ok {
				b.Skip("DeepCopyMap has quadratic running time.")
			}
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				j := i % (hits + misses)
				if j < hits {
					if _, ok := m.LoadOrStore(j, i); !ok {
						b.Fatalf("unexpected miss for %v", j)
					}
				} else {
					if v, loaded := m.LoadOrStore(i, i); loaded {
						b.Fatalf("failed to store %v: existing value %v", i, v)
					}
				}
			}
		},
	})
}

func BenchmarkLoadOrStoreUnique(b *testing.B) {
	benchMap(b, bench[int, int]{
		setup: func(b *testing.B, m mapInterface[int, int]) {
			if _, ok := m.(*DeepCopyMap[int, int]); ok {
				b.Skip("DeepCopyMap has quadratic running time.")
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.LoadOrStore(i, i)
			}
		},
	})
}

func BenchmarkLoadOrStoreCollision(b *testing.B) {
	benchMap(b, bench[int, int]{
		setup: func(_ *testing.B, m mapInterface[int, int]) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.LoadOrStore(0, 0)
			}
		},
	})
}

func BenchmarkLoadAndDeleteBalanced(b *testing.B) {
	const hits, misses = 128, 128

	benchMap(b, bench[int, int]{
		setup: func(b *testing.B, m mapInterface[int, int]) {
			if _, ok := m.(*DeepCopyMap[int, int]); ok {
				b.Skip("DeepCopyMap has quadratic running time.")
			}
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				j := i % (hits + misses)
				if j < hits {
					m.LoadAndDelete(j)
				} else {
					m.LoadAndDelete(i)
				}
			}
		},
	})
}

func BenchmarkLoadAndDeleteUnique(b *testing.B) {
	benchMap(b, bench[int, int]{
		setup: func(b *testing.B, m mapInterface[int, int]) {
			if _, ok := m.(*DeepCopyMap[int, int]); ok {
				b.Skip("DeepCopyMap has quadratic running time.")
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.LoadAndDelete(i)
			}
		},
	})
}

func BenchmarkLoadAndDeleteCollision(b *testing.B) {
	benchMap(b, bench[int, int]{
		setup: func(_ *testing.B, m mapInterface[int, int]) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.LoadAndDelete(0)
			}
		},
	})
}

func BenchmarkRange(b *testing.B) {
	const mapSize = 1 << 10

	benchMap(b, bench[int, int]{
		setup: func(_ *testing.B, m mapInterface[int, int]) {
			for i := 0; i < mapSize; i++ {
				m.Store(i, i)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.Range(func(_, _ int) bool { return true })
			}
		},
	})
}

// BenchmarkAdversarialAlloc tests performance when we store a new value
// immediately whenever the map is promoted to clean and otherwise load a
// unique, missing key.
//
// This forces the Load calls to always acquire the map's mutex.
func BenchmarkAdversarialAlloc(b *testing.B) {
	benchMap(b, bench[int, int64]{
		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int64]) {
			var stores, loadsSinceStore int64
			for ; pb.Next(); i++ {
				m.Load(i)
				if loadsSinceStore++; loadsSinceStore > stores {
					m.LoadOrStore(i, stores)
					loadsSinceStore = 0
					stores++
				}
			}
		},
	})
}

// BenchmarkAdversarialDelete tests performance when we periodically delete
// one key and add a different one in a large map.
//
// This forces the Load calls to always acquire the map's mutex and periodically
// makes a full copy of the map despite changing only one entry.
func BenchmarkAdversarialDelete(b *testing.B) {
	const mapSize = 1 << 10

	benchMap(b, bench[int, int]{
		setup: func(_ *testing.B, m mapInterface[int, int]) {
			for i := 0; i < mapSize; i++ {
				m.Store(i, i)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.Load(i)

				if i%mapSize == 0 {
					m.Range(func(k, _ int) bool {
						m.Delete(k)
						return false
					})
					m.Store(i, i)
				}
			}
		},
	})
}

func BenchmarkDeleteCollision(b *testing.B) {
	benchMap(b, bench[int, int]{
		setup: func(_ *testing.B, m mapInterface[int, int]) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m mapInterface[int, int]) {
			for ; pb.Next(); i++ {
				m.Delete(0)
			}
		},
	})
}
