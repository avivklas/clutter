package clutter

import (
	"fmt"
	"slices"
)

// Clutter is a rather stupid idea to solve small-size cluttered application
// data represented by sparse lists of structs in memory. you may think of it as
// a simple, primitive database of all the data as structs in memory.
// it was built to consolidate caching layer operations where the cache layer is
// the only access layer for access. no cold layer. data is always prepared on
// the hot cache. it is very inefficient in writes, but it's ok in reads.
// the only advantage of this structure is that it let you manage your data in a
// very naive and straight forward manner. it is therefore meets the needs of
// application configuration data that is usually not big. it is expected that
// k <= 10000. for larger data sets, other in-mem relational dbs will be a
// better choice for you.
type Clutter interface {
	// ReloadAll calls all loaders to load all their data into the db
	ReloadAll()

	// Invalidate invalidates the specified items and reloads all the affected
	// items of the kinds of the provided items and their related kinds
	Invalidate(items ...Item)
}

// New creates a builder of clutter
func New() builder {
	return builder{}
}

type builder map[string]loader

func (b builder) WithLoader(kind string, loaderFunc Loader) builder {
	b[kind] = loader{
		kind: kind,
		load: loaderFunc,
	}

	return b
}

func (b builder) Clutter(db DB) Clutter {
	return &clutter{
		cache:   db,
		loaders: b,
	}
}

type Loader func(add func(...Item))

type loader struct {
	kind string
	load Loader
}

type clutter struct {
	cache   DB
	loaders map[string]loader
}

func (cl *clutter) ReloadAll() {
	_ = cl.cache.Update(func(writer DBWriter) error {
		for _, l := range cl.loaders {
			cl.reload(writer, l)
		}

		return nil
	})
}

func (cl *clutter) reload(writer DBWriter, loader loader) {
	loader.load(func(items ...Item) {
		for _, item := range items {
			index(item, func(item Item, key string, tags ...string) {
				writer.Put(key, item)
				if len(tags) > 0 {
					writer.Tag(key, tags...)
					for _, tag := range tags {
						writer.Tag(tag, key)
					}
				}
			})
		}
	})
}

func (cl *clutter) Query(kind string, tag Pointer, collect func(interface{}) bool) (err error) {
	cl.cache.Iter(tag.String(), func(key string, getVal func() (interface{}, bool)) (proceed bool) {
		var (
			p    Pointer
			item interface{}
			ok   bool
		)

		err = p.Parse(key)
		if err != nil {
			return false
		}

		if p.Kind != kind || p.Key != tag.Key {
			return true
		}

		item, ok = getVal()
		if !ok {
			err = fmt.Errorf("failed to retrieve value of %q", key)
			return false
		}

		return collect(item)
	})

	return
}

func (cl *clutter) Invalidate(items ...Item) {
	var forDeletion []string
	for _, item := range items {
		for _, i := range item.Indexes() {
			forDeletion = append(forDeletion, key(item.Kind(), i[0], i[1]))
		}
	}

	_ = cl.cache.Update(func(writer DBWriter) error {
		deleted := writer.Invalidate(forDeletion...)

		var (
			p        Pointer
			reloaded []string
		)
		for _, deletedKey := range deleted {
			_ = p.Parse(deletedKey)
			if !slices.Contains(reloaded, p.Kind) {
				if loader, ok := cl.loaders[p.Kind]; ok {
					cl.reload(writer, loader)
					reloaded = append(reloaded, p.Kind)
				}
			}
		}

		return nil
	})
}
