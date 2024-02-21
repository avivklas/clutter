package clutter

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type Item interface {
	Indexes() [][2]string
	Relations() []Item
	Kind() string
}

type Pointer struct {
	Kind, Key, Val string
}

const (
	curlyStart = '{'
	curlyEnd   = '}'
	colon      = ':'
)

var bufPool = sync.Pool{New: func() any {
	return &bytes.Buffer{}
}}

func key(kind, key, val string) (res string) {
	buf := bufPool.Get().(*bytes.Buffer)
	writeKey(buf, kind, key, val)
	res = buf.String()
	buf.Reset()
	bufPool.Put(buf)

	return
}

func writeKey(dst *bytes.Buffer, kind, key, val string) {
	dst.WriteString(kind)
	dst.WriteRune(curlyStart)
	dst.WriteString(key)
	dst.WriteRune(colon)
	dst.WriteString(val)
	dst.WriteRune(curlyEnd)
}

func (p *Pointer) String() string {
	return key(p.Kind, p.Key, p.Val)
}

func (p *Pointer) Parse(key string) error {
	parts := strings.SplitN(strings.Trim(key, "}"), "{", 2)
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse %q as pointer. expected format: 'kind{key:val}'", key)
	}

	keyVal := strings.SplitN(parts[1], ":", 2)
	if len(keyVal) != 2 {
		return fmt.Errorf("failed to parse %q as pointer. expected format: 'kind{key:val}'", key)
	}

	p.Kind, p.Key, p.Val = parts[0], keyVal[0], keyVal[1]

	return nil
}

func index(in Item, store func(item Item, key string, tags ...string), tags ...string) {
	if in == nil || reflect.ValueOf(in).IsNil() {
		return
	}

	indexes := in.Indexes()
	relationTags := make([]string, len(indexes)+len(tags))
	for i, p := range indexes {
		relationTags[i] = key(in.Kind(), p[0], p[1])
		store(in, relationTags[i], tags...)
	}

	for i := 0; i < len(tags); i++ {
		relationTags[len(indexes)+i] = tags[i]
	}

	for _, related := range in.Relations() {
		index(related, store, relationTags...)
	}

	return
}
