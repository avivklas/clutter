package clutter

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

type meta struct {
	id   string
	name string
}

func (m meta) Indexes() [][2]string {
	return [][2]string{{"id", m.id}, {"name", m.name}}
}

type fooItem struct {
	meta

	fooValue string
}

func (fooItem) Kind() string {
	return "foo"
}

func (f fooItem) Relations() (res []Item) {

	return
}

type barItem struct {
	meta

	barValue string
	foos     []*fooItem
}

func (barItem) Kind() string {
	return "bar"
}

func (b barItem) Relations() (res []Item) {
	for _, foo := range b.foos {
		res = append(res, foo)
	}

	return
}

func Test_clutter(t *testing.T) {
	db := NewDB()

	foos := []*fooItem{{
		meta:     meta{"1", "foo1"},
		fooValue: "I'm foo",
	}, {
		meta:     meta{"2", "foo2"},
		fooValue: "I'm foo #2",
	}}

	bars := []*barItem{{
		meta:     meta{"1", "bar1"},
		barValue: "I'm bar",
		foos:     foos,
	}}

	cl := New().
		WithLoader("bar", func(add func(...Item)) {
			for _, bar := range bars {
				add(bar)
			}
		}).
		Clutter(db)

	cl.ReloadAll()

	getFooById := MkGetter[*fooItem](db, "id")

	foo, ok := getFooById("1")
	assert.True(t, ok)
	assert.Equal(t, "I'm foo", foo.fooValue)

	getBar := MkGetter[*barItem](db, "id")
	barsByFooId := MkQueryByRelation[*barItem](db, "id", "foo", "id")

	bar, ok := getBar("1")
	assert.True(t, ok)
	assert.Equal(t, "I'm bar", bar.barValue)

	barList, err := barsByFooId("1")
	if !assert.NoError(t, err) {
		return
	}

	cl.Invalidate(fooItem{meta: meta{id: "1"}})
	assert.Len(t, barList, 1)

	foosByBarId := MkQueryByRelation[*fooItem](db, "id", "bar", "id")

	fooList, err := foosByBarId("1")
	if !assert.NoError(t, err) {
		return
	}

	assert.Len(t, fooList, 2)

	foos[0].fooValue = "this foo has changed"
	bars[0].barValue = "this bar has changed"

	cl.Invalidate(foo)

	foo2, ok := getFooById("2")
	assert.True(t, ok)
	assert.Equal(t, "I'm foo #2", foo2.fooValue)

	bar, ok = getBar("1")
	assert.True(t, ok)
	assert.Equal(t, "this bar has changed", bar.barValue)

	foo, ok = getFooById("1")
	assert.True(t, ok)
	assert.Equal(t, "this foo has changed", foo.fooValue)

}

/*
benchmark result as first committed the solution on a MacBook Pro 2020 model

goos: darwin
goarch: amd64
pkg: github.com/avivklas/clutter
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz
Benchmark_clutter
Benchmark_clutter/get
Benchmark_clutter/get-8                             3702513      318.1 ns/op
Benchmark_clutter/query_one-to-one_relation
Benchmark_clutter/query_one-to-one_relation-8        708255	      1730 ns/op
Benchmark_clutter/query_one-to-many_relation
Benchmark_clutter/query_one-to-many_relation-8       406094       3010 ns/op
*/
func Benchmark_clutter(b *testing.B) {
	db := NewDB()

	b.SetParallelism(1)

	var foos []Item
	for i := 0; i < 1000; i++ {
		foos = append(foos, &fooItem{
			meta: meta{fmt.Sprintf("%d", i), fmt.Sprintf("foo %d", i)},
		})
	}

	var bars []Item
	for i := 0; i < 1000/4; i++ {
		var foos []*fooItem
		for j := 0; j < 4; j++ {
			foos = append(foos, &fooItem{meta: meta{fmt.Sprintf("%d", i*4+j), fmt.Sprintf("bar %d", i*4+j)}})
		}
		bars = append(bars, &barItem{
			meta: meta{fmt.Sprintf("%d", i), fmt.Sprintf("bar %d", i)},
			foos: foos,
		})
	}

	cl := New().
		//WithLoader("foo", func(add func(...Item)) { add(foos...) }).
		WithLoader("bar", func(add func(...Item)) { add(bars...) }).
		Clutter(db)

	cl.ReloadAll()

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	b.Run("get", func(b *testing.B) {
		getFoo := MkGetter[*fooItem](db, "id")
		var ok bool
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("%d", rnd.Int63n(1000))
			_, ok = getFoo(key)
			if !assert.True(b, ok, key) {
				return
			}
		}
	})

	b.Run("query one-to-one relation", func(b *testing.B) {
		barsByFooId := MkQueryByRelation[*barItem](db, "id", "foo", "id")
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			barList, err := barsByFooId(fmt.Sprintf("%d", rnd.Int63n(1000)))
			if !assert.NoError(b, err) {
				b.Fatal(err)
			}

			assert.Len(b, barList, 1)
		}
	})

	b.Run("query one-to-many relation", func(b *testing.B) {
		foosByBarId := MkQueryByRelation[*fooItem](db, "id", "bar", "id")
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			fooList, err := foosByBarId(fmt.Sprintf("%d", rnd.Int63n(1000/4)))
			if !assert.NoError(b, err) {
				b.Fatal(err)
			}

			assert.Len(b, fooList, 4)
		}
	})

	b.Run("invalidate", func(b *testing.B) {
		cl.ReloadAll()
		foo := &fooItem{meta: meta{id: "333"}}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cl.Invalidate(foo)
		}
	})
}
