// +build unit
// +build !integration

package collection

import (
	"fmt"
	"testing"
)

func ExampleUniqueNames() {
	var strings = []string{
		"a",
		"b",
		"a",
		"a",
		"b",
	}
	uniqueStrings := UniqueNames(strings)

	for _, s := range uniqueStrings {
		fmt.Println(s)
	}

	// Unordered output:
	// a
	// b
}

func ExampleUniqueIDs() {
	var nums = []int{
		1,
		2,
		1,
		1,
		2,
		3,
		1,
	}
	uniqueNums := UniqueIDs(nums)

	for _, n := range uniqueNums {
		fmt.Println(n)
	}

	// Unordered output:
	// 1
	// 2
	// 3
}

func TestFindReplace(t *testing.T) {
	var terms = [...]string{
		"xx",
		"ab",
		"bc",
	}
	var mapping = map[string]*string{
		"test34": &terms[0],
		"test1":  &terms[1],
		"test2":  &terms[2],
	}
	var mapping2 = map[string][]*string{
		"test34": []*string{&terms[0]},
		"test1":  []*string{&terms[1]},
		"test2":  []*string{&terms[2]},
	}

	t.Run("strict", func(t *testing.T) {
		t.Parallel()
		t1, _ := StrictFindReplace("test1", mapping)
		t2, _ := StrictFindReplace("test2", mapping)
		t3, _ := StrictFindReplace("test3", mapping)

		if t1 != "ab" || t2 != "bc" || t3 != "" {
			t.Fatalf("Failed to findreplace")
		}
	})

	t.Run("fuzzy", func(t *testing.T) {
		t.Parallel()
		t1, _ := FuzzyFindReplace("test1", mapping)
		t2, _ := FuzzyFindReplace("test2", mapping)
		t3, _ := FuzzyFindReplace("test3", mapping)

		if t1 != "ab" || t2 != "bc" || t3 != "test3" {
			t.Fatalf("Failed to findreplace")
		}
	})

	t.Run("strict2", func(t *testing.T) {
		t.Parallel()
		t1, _ := StrictFindReplace2("test1", mapping2)
		t2, _ := StrictFindReplace2("test2", mapping2)
		t3, _ := StrictFindReplace2("test3", mapping2)

		if len(t1) == 0 || len(t2) == 0 || len(t3) > 0 {
			t.Fatalf("Failed to findreplace")
		}
	})

	t.Run("fuzzy2", func(t *testing.T) {
		t.Parallel()
		t1, _ := FuzzyFindReplace2("test1", mapping2)
		t2, _ := FuzzyFindReplace2("test2", mapping2)
		t3, _ := FuzzyFindReplace2("test3", mapping2)

		if len(t1) == 0 || len(t2) == 0 || len(t3) == 0 {
			t.Fatalf("Failed to findreplace")
		}
	})
}

func ExampleSanitize() {
	txt := "A+B@C.D!E"

	t1 := Sanitize(txt)
	t2 := SanitizeHard(txt)

	fmt.Println(t1)
	fmt.Println(t2)

	// Output:
	// A+B@C.D!E
	// a b c d e
}

func ExampleRemoveFromString() {
	var selectors = []string{
		"test",
		"-",
	}
	fmt.Println(RemoveFromString("Hello - test", selectors))

	// Output:
	// Hello
}

func BenchmarkCollections(b *testing.B) {
	var strings = []string{
		"a",
		"b",
		"a",
		"a",
		"b",
	}
	var nums = []int{
		1,
		2,
		1,
		1,
		2,
		3,
		1,
	}

	var m0 = map[string]string{
		"test1": "ab",
		"test2": "bc",
	}

	mapping := make(map[string]*string, 2)
	for k := range m0 {
		v := m0[k]
		mapping[k] = &v
	}

	for i := 0; i < b.N; i++ {
		UniqueNames(strings)
		UniqueIDs(nums)
		FuzzyFindReplace("test1", mapping)
		StrictFindReplace("test1", mapping)
	}
}

func prepareMapAttributes() (mapping map[string][]*string, instrings []string) {
	var terms = []string{
		"red",
		"orange",
		"blue",
		"green",
	}
	mapping = map[string][]*string{
		"fire": []*string{
			&terms[0],
			&terms[1],
		},
		"water": []*string{
			&terms[2],
			&terms[3],
		},
	}
	instrings = []string{
		"firefly",
		"waterman",
		"lemongras",
	}

	return mapping, instrings
}

func ExampleMapAttributesStrict() {
	mapping, instrings := prepareMapAttributes()
	attributes, err := MapAttributes(instrings, mapping, "", true)
	if err != nil {
		return
	}

	for i := range attributes {
		fmt.Println(attributes[i])
	}

	// Unordered output:
	// red
	// orange
	// blue
	// green
	//
}

func ExampleMapAttributes() {
	mapping, instrings := prepareMapAttributes()
	attributes, err := MapAttributes(instrings, mapping, "", false)
	if err != nil {
		return
	}

	for i := range attributes {
		fmt.Println(attributes[i])
	}

	// Unordered output:
	// red
	// orange
	// blue
	// green
	// lemongras
}

func TestMapAttributes(t *testing.T) {
	var terms = []string{
		"stork",
		"flubb",
	}
	mapping, _ := prepareMapAttributes()
	attributes, err := MapAttributes(terms, mapping, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(attributes) != 2 {
		t.Fatalf("Should have returned two terms - %v", attributes)
	}

	attributes, err = MapAttributes(terms, mapping, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(attributes) > 0 {
		t.Fatalf("Shouldn't have matched test terms - %v", attributes)
	}
}

func TestCollateStrings(t *testing.T) {
	s := CollateStrings("", "a", "", "b")
	if s != "a" {
		t.Fatalf("Failed to match the correct string - %s instead of a", s)
	}
}
