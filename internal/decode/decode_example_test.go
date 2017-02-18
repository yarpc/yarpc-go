package decode

import (
	"fmt"
	"log"
	"sort"
)

func ExampleDecode() {
	type Item struct {
		Key   string `config:"name"`
		Value string
	}

	var item Item
	err := Decode(&item, map[string]interface{}{
		"name":  "foo",
		"value": "bar",
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(item.Key, item.Value)
	// Output: foo bar
}

type StringSet map[string]struct{}

func (ss *StringSet) Decode(decode Into) error {
	var items []string
	if err := decode(&items); err != nil {
		return err
	}

	*ss = make(map[string]struct{})
	for _, item := range items {
		(*ss)[item] = struct{}{}
	}
	return nil
}

func ExampleDecode_decoder() {
	var ss StringSet

	err := Decode(&ss, []interface{}{"foo", "bar", "foo", "baz"})
	if err != nil {
		log.Fatal(err)
	}

	var items []string
	for item := range ss {
		items = append(items, item)
	}
	sort.Strings(items)

	fmt.Println(items)
	// Output: [bar baz foo]
}
