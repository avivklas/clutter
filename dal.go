package clutter

import (
	"fmt"
	"reflect"
)

// Getter is a function for fetching 1 item of concrete type by a specific key
type Getter[T Item] func(val string) (T, bool)

// MkGetter creates a Getter for a concrete type by a specific key
func MkGetter[T Item](db DB, byKey string) Getter[T] {
	kind, err := itemKind[T]()
	if err != nil {
		panic(err)
	}

	return func(val string) (t T, ok bool) {
		i, ok := db.Get(key(kind, byKey, val))
		if !ok {
			return
		}

		t, ok = i.(T)

		return
	}
}

// Query is a function for fetching list of items of a concrete type by tags
type Query[T Item] func(string) ([]T, error)

// MkQueryByRelation creates a Query bound to fetching items of T that are
// related to another kind.
//
// uniqueKey is the key that is used to distinct resulted items
// otherKind is the kind of the relation
// otherKey is the key that you wish to query by
//
// for example: T = *foo, uniqueKey = "id", otherKind = "bar", otherKey = "name"
// will create a query function that lets you query all "foos" that are related
// to bar with a specified name.
func MkQueryByRelation[T Item](db DB, uniqueKey, otherKind, otherKey string) Query[T] {
	kind, err := itemKind[T]()
	if err != nil {
		panic(err)
	}

	return func(val string) (res []T, err error) {
		db.Iter(key(otherKind, otherKey, val), func(key string, getVal func() (interface{}, bool)) (proceed bool) {
			var (
				p    Pointer
				item interface{}
				t    T
				ok   bool
			)

			err = p.Parse(key)
			if err != nil {
				return false
			}

			if p.Kind != kind {
				return true
			}

			if p.Key != uniqueKey {
				return true
			}

			item, ok = getVal()
			if !ok {
				err = fmt.Errorf("failed to retrieve value of %q", key)
				return false
			}

			t, ok = item.(T)
			if !ok {
				err = fmt.Errorf("expected type %T for %q. got %T", t, key, item)
				return false
			}

			res = append(res, t)

			return true
		})

		return
	}
}

func itemKind[T Item]() (string, error) {
	var t T
	tt := reflect.TypeOf(t)
	if tt.Kind() == reflect.Ptr {
		tt = tt.Elem()
	}

	tFactory, ok := reflect.New(tt).Interface().(Item)
	if !ok {
		return "", fmt.Errorf("failed to infer kind of %T", t)
	}

	return tFactory.Kind(), nil
}
