package clutter

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_db(t *testing.T) {
	c := NewDB()
	assertKV := func(k, v string) {
		val, ok := c.Get(k)
		if !assert.True(t, ok, "key [%s] should exist", k) {
			t.Fatal()
		}

		assert.Equal(t, v, val, "expected value at [%s] to equal [%s]", k, v)
	}

	c.Put("foo", "bar")
	c.Tag("foo", "1", "2", "3")
	assertKV("foo", "bar")

	c.Put("bar", "baz")
	c.Tag("bar", "1", "2", "3")
	assertKV("bar", "baz")

	expectedResult := "foobarbarbaz"
	var actualResult string
	c.Iter("1", func(key string, getVal func() (interface{}, bool)) (proceed bool) {
		actualResult += key
		v, ok := getVal()
		assert.True(t, ok)

		s, ok := v.(string)

		actualResult += s

		return true
	})
	assert.Equal(t, expectedResult, actualResult)

	c.Invalidate("4")
	assertKV("bar", "baz")

	c.Invalidate("3")
	_, ok := c.Get("foo")
	if !assert.False(t, ok) {
		return
	}

	c.Put("foo", "bar")
	c.Tag("foo", "x")
	c.Invalidate("1")
	assertKV("foo", "bar")
	c.Invalidate("x")

	val, ok := c.GetOrFill("foo", func() interface{} {
		return nil
	})
	if !assert.False(t, ok) {
		return
	}
	assert.Equal(t, val, nil)

	val, ok = c.GetOrFill("foo", func() interface{} {
		return "barz"
	})
	if !assert.True(t, ok) {
		return
	}

	assert.Equal(t, val, "barz")
	assertKV("foo", "barz")

	val, ok = c.GetOrFill("foo", func() interface{} {
		return "bar"
	})
	if !assert.True(t, ok) {
		return
	}

	assert.Equal(t, val, "barz")
	assertKV("foo", "barz")
}

/*
benchmark result as first committed the solution on a MacBook Pro 2020 model

goos: darwin
goarch: amd64
pkg: github.com/avivklas/clutter
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz
Benchmark_db
Benchmark_db/write
Benchmark_db/write-8               202022            5932 ns/op       17424 B/op          12 allocs/op
Benchmark_db/read
Benchmark_db/read-8              11614705           103.1 ns/op          24 B/op           2 allocs/op
Benchmark_db/invalidate
Benchmark_db/invalidate-8         8859303           137.1 ns/op          32 B/op           3 allocs/op
*/
func Benchmark_db(b *testing.B) {
	initDB := func(n int) DB {
		db := NewDB()
		for i := 0; i < n; i++ {
			db.Put(fmt.Sprintf("key:%d", i), struct{}{})
		}

		return db
	}

	b.Run("write", func(b *testing.B) {
		db := initDB(64)
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			val := i % 64
			db.Put(fmt.Sprintf("key:%d", val), struct{}{})
		}
	})

	b.Run("read", func(b *testing.B) {
		db := initDB(64)

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			db.Get(fmt.Sprintf("key:%d", i))
		}
	})

	b.Run("invalidate", func(b *testing.B) {
		db := initDB(64)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			db.Invalidate(fmt.Sprintf("%d", i))
		}
	})
}
