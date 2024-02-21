# Clutter

Clutter is a rather stupid idea to solve small-size cluttered application
data represented by sparse lists of structs in memory. You may think of it as
a simple, primitive database of all the data as structs in memory.

It was built to consolidate caching layer operations where the cache layer is
the only access layer for access. No cold layer. Data is always prepared on
the hot cache. It is very inefficient in writes, but it's ok in reads.

The only advantage of this structure is that it let you manage your data in a
very naive and straight forward manner. It is therefore meets the needs of
application configuration data that is usually not big. It is expected that
k <= 10000. For larger data sets, other in-mem relational dbs will be a better
choice for you.

## Components

### `DB`

a primitive storage layer  
how to init:
```go
db := DB()
```

### `Item`
every item that clutter manages must implement this interface.  
**example:**
```go
type foo struct {
	id   string
	name string
	bars []*bar
}

func (i item) Kind() string {
    return "foo"
}

func (i item) Indexes() [][2]string {
    return [][2]string{{"id", i.id}, {"name", i.name}}
}

func (i item) Relations() (relations []Item) {
    for _ b := range i.bars {
        relations = append(relations, b)
    }
}
```

### `Loader` (multiple)

a simple func that you implement in order to load a specific kind to clutter.  
here's an example of loading `foo` from an SQL db:
```go
l := Loader(func(add ...Item) {
    rows, err := db.Query("select id, name from foo")
    if err != nil {
        return
    }
    defer rows.Close()
	
    var foo foo
    for rows.Next() {
        err = rows.Scan(&foo)
        if err != nil {
            return
        }
		
        add(&foo)
    }
})
```

### Init Clutter

```go
import "github.com/avivklas/clutter"

db := clutter.DB()

cl := clutter.New().
	WithLoader("foo", fooLoader).
	Clutter(db)
```

### Reload Data
reloading the data is performed as a reaction to invalidation of an item. it
deletes all related items and reloads all the relevant kinds (currently all
data of a kind, not only the invalidated items).
```go
cl.Invalidate(fooItem{meta: meta{id: "1"}})
```

##  DAL
```go
import "github.com/avivklas/clutter"

// a method for fetching foo by its id
getFooById := clutter.MkGetter[*foo](db, "id")

// a method for querying all bars related to foo by foo id
barsByFooId := clutter.MkQueryByRelation[*bar](db, "id", "foo", "id")
```

## Performance
```shell
goos: darwin
goarch: amd64
pkg: github.com/avivklas/clutter
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz
Benchmark_clutter
Benchmark_clutter/get
Benchmark_clutter/get-8                              	 3857265	       279 ns/op	      43 B/op	       3 allocs/op
Benchmark_clutter/query_one-to-one_relation
Benchmark_clutter/query_one-to-one_relation-8         	  685304	      1671 ns/op	     404 B/op	      14 allocs/op
Benchmark_clutter/query_one-to-many_relation
Benchmark_clutter/query_one-to-many_relation-8        	  375578	      3172 ns/op	    1113 B/op	      33 allocs/op
Benchmark_clutter/invalidate
Benchmark_clutter/invalidate-8                        	     361	   3185767 ns/op	  293169 B/op	    9769 allocs/op
```