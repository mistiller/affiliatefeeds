package collection

import (
	"fmt"
	"hash/fnv"
	"html"
	"regexp"
	"strconv"
	"strings"
)

// UniqueNames returns a slice of unique elements of in
func UniqueNames(in []string) []string {
	var out []string
	uniqueMap := make(map[string]struct{})
	for i := range in {
		if in[i] == "" {
			continue
		}

		_, exist := uniqueMap[in[i]]
		if exist == true {
			continue
		}
		uniqueMap[in[i]] = struct{}{}
	}

	out = make([]string, len(uniqueMap))
	var i int
	for k := range uniqueMap {
		out[i] = k
		i++
	}

	return out
}
func UniqueIDs(in []int) []int {
	var out []int
	uniqueMap := make(map[int]struct{})
	for i := range in {
		_, exist := uniqueMap[in[i]]
		if exist == true {
			continue
		}
		uniqueMap[in[i]] = struct{}{}
	}

	out = make([]int, len(uniqueMap))
	var i int
	for k := range uniqueMap {
		out[i] = k
		i++
	}

	return out
}
func UniqueUint64(in []uint64) []uint64 {
	var out []uint64
	uniqueMap := make(map[uint64]struct{})
	for i := range in {
		_, exist := uniqueMap[in[i]]
		if exist == true {
			continue
		}
		uniqueMap[in[i]] = struct{}{}
	}

	out = make([]uint64, len(uniqueMap))
	var i int
	for k := range uniqueMap {
		out[i] = k
		i++
	}

	return out
}

// FuzzyFindReplace looks for the occurence of a key and returns "" if no match
func FuzzyFindReplace(s string, mapping map[string]*string) (replacement string, matched bool) {
	var (
		thisMatched  bool
		q            string
		longestMatch int
	)
	replacement = s
	for key := range mapping {
		q = fmt.Sprintf(".*%s", key)
		thisMatched, _ = regexp.MatchString(q, s)
		if thisMatched == true {
			matched = true
			if len(key) > longestMatch {
				replacement = *mapping[key]
				longestMatch = len(key)
			}
		}
	}
	return replacement, matched
}

// StrictFindReplace looks for the occurence of a key and returns "" if no match
func StrictFindReplace(instring string, mapping map[string]*string) (replacement string, matched bool) {
	var thisMatched bool
	var q string
	var longestMatch int
	for key := range mapping {
		q = fmt.Sprintf(".*%s", key)
		thisMatched, _ = regexp.MatchString(q, instring)
		if thisMatched == true {
			matched = true
			if len(key) > longestMatch {
				replacement = *mapping[key]
				longestMatch = len(key)
			}
		}
	}
	return replacement, matched
}

// FuzzyFindReplace2 looks for the occurence of a key and returns "" if no match
func FuzzyFindReplace2(instring string, mapping map[string][]*string) (replacements []string, matched bool) {
	var (
		thisMatched  bool
		q            string
		longestMatch int
	)

	s := Sanitize(strings.ToLower(instring))
	replacements = []string{
		s,
	}
	for key := range mapping {
		q = fmt.Sprintf(".*%s", key)
		thisMatched, _ = regexp.MatchString(q, s)
		if thisMatched == true {
			matched = true
			if len(key) > longestMatch {
				for k := range mapping[key] {
					replacements = append(replacements, *mapping[key][k])
				}
				longestMatch = len(key)
			}
		}
	}

	return UniqueNames(replacements), matched
}

// StrictFindReplace2 looks for the occurence of a key and returns "" if no match
func StrictFindReplace2(instring string, mapping map[string][]*string) (replacements []string, matched bool) {
	var (
		thisMatched  bool
		q            string
		longestMatch int
	)

	s := Sanitize(strings.ToLower(instring))

	for key := range mapping {
		q = fmt.Sprintf(".*%s", key)
		thisMatched, _ = regexp.MatchString(q, s)
		if thisMatched == true {
			matched = true
			if len(key) > longestMatch {
				for k := range mapping[key] {
					replacements = append(replacements, *mapping[key][k])
				}
				longestMatch = len(key)
			}
		}
	}
	return UniqueNames(replacements), matched
}

func Sanitize(s string) (str string) {
	str = html.UnescapeString(strings.TrimSpace(s))
	var replacements = [...]string{
		"\"",
		"#",
		"*",
		"_",
		"\n",
		"\r",
	}

	for i := range replacements {
		str = strings.Replace(str, replacements[i], "", -1)
	}

	return strings.TrimSpace(str)
}
func SanitizeHard(s string) string {
	s = html.UnescapeString(strings.TrimSpace(s))

	reg, _ := regexp.Compile("[^a-zA-Z]+")
	s = reg.ReplaceAllString(strings.ToLower(s), " ")

	return strings.TrimSpace(s)
}
func SplitList(s string) (out []string) {
	split := func(r rune) bool {
		return r == ',' || r == ':' || r == ';' || r == '.' || r == '/' || r == '#' || r == '>'
	}
	out = strings.FieldsFunc(s, split)
	for i := 0; i < len(out); i++ {
		out[i] = Sanitize(out[i])
	}
	return out
}
func HashKey(s string) uint64 {
	s = SanitizeHard(s)
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}
func AnyIsNil(s []string) bool {
	for i := range s {
		if s[i] == "" {
			return true
		}
	}
	return false
}

// CollateString returns a if a != nil, else b
func CollateString(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

// CollateStrings returns a the first non-empty string in list
func CollateStrings(input ...string) string {
	for i := range input {
		if input[i] != "" {
			return input[i]
		}
	}
	return ""
}

// CollateInt returns a if a != nil, else b
func CollateInt(a, b int32) int32 {
	if a == 0 {
		return b
	}
	return a
}

// CollateFloat returns a if a != nil, else b
func CollateFloat(a, b float32) float32 {
	if a == 0.0 {
		return b
	}
	return a
}
func RemoveElement(s []interface{}, i int) []interface{} {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}

// StringInList returns true if a given string is in a list of strings
func StringInList(str string, list []string) bool {
	for i := range list {
		if str == list[i] {
			return true
		}
	}
	return false
}

// ListInList returns true if a given string is in a list of strings
func ListInList(l0 []string, l1 []string) bool {
	for i := range l0 {
		for j := range l1 {
			if l0[i] == l1[j] {
				return true
			}
		}
	}
	return false
}

// RemoveFromString cleans a list of substrings out of a string
// and tries not to leave any spaces behind
func RemoveFromString(str string, selectors []string) string {
	for n := range selectors {
		if strings.Contains(str, selectors[n]) {
			str = strings.Replace(str, selectors[n], "", -1)
			str = strings.Trim(str, " ")
			str = strings.Trim(str, ",")
			str = strings.Trim(str, "-")
			str = strings.Trim(str, ";")
		}
	}
	return str
}

// IsEmpty checks for empty string
func IsEmpty(s *string) bool {
	return *s == ""
}

// AnyEmpty checks for any empty string in slice
func AnyEmpty(s []*string) bool {
	for i := range s {
		if *s[i] == "" {
			return true
		}
	}
	return false
}

// AnyEmptyFloats checks for any empty float in slice
func AnyEmptyFloats(s []*float32) bool {
	for i := range s {
		if *s[i] == 0.0 {
			return true
		}
	}
	return false
}

// MergeLists merges two string slices
func MergeLists(a []string, b []string) []string {
	c := make([]string, len(a)+len(b))
	idx := 0
	for i := range a {
		c[idx] = a[i]
		idx++
	}
	for j := range b {
		c[idx] = b[j]
		idx++
	}
	return c
}

func HighestFloat(a []float32) (c float32) {
	if len(a) < 1 {
		return 0.0
	}
	var highest float32
	for i := range a {
		if a[i] > highest {
			highest = a[i]
		}
	}
	return highest
}

func LowestFloat(a []float32) (c float32) {
	if len(a) < 1 {
		return 0.0
	}
	lowest := HighestFloat(a)
	for i := range a {
		if a[i] < lowest {
			lowest = a[i]
		}
	}
	return lowest
}

func MapAttributes(instrings []string, mapping map[string][]*string, fallback string, strict bool) (attributes []string, err error) {
	var (
		arr                                              []string
		s                                                string
		termMatched, candidateMatched, anyMatched, exist bool
	)

	candidates := make(map[string]struct{})

	var str []string
	for i := range instrings {
		candidateMatched = false

		arr = SplitList(instrings[i])
		for j := range arr {
			s = strings.ToLower(Sanitize(arr[j]))

			_, exist = candidates[s]
			if exist {
				continue
			}
			candidates[s] = struct{}{}

			str, termMatched = StrictFindReplace2(s, mapping)

			if termMatched {
				candidateMatched = true
				anyMatched = true
				for k := range str {
					attributes = append(attributes, strings.ToLower(str[k]))
				}
			}
		}
		if !candidateMatched && !strict {
			attributes = append(attributes, instrings[i])
			anyMatched = true
		}
	}

	if !anyMatched && fallback != "" {
		attributes = append(attributes, fallback)
		anyMatched = true
	}

	/*if !anyMatched {
		return attributes, fmt.Errorf("Failed to map candidates: %v", candidates)
	}*/

	return UniqueNames(attributes), nil
}

func MapToCSV(m map[string]interface{}) []string {
	values := make([]string, 0, len(m))
	for _, v := range m {
		switch vv := v.(type) {
		case map[string]interface{}:
			for _, value := range MapToCSV(vv) {
				values = append(values, value)
			}
		case string:
			values = append(values, vv)
		case float64:
			values = append(values, strconv.FormatFloat(vv, 'f', -1, 64))
		case []interface{}:
			// Arrays aren't currently handled, since you haven't indicated that we should
			// and it's non-trivial to do so.
		case bool:
			values = append(values, strconv.FormatBool(vv))
		case nil:
			values = append(values, "nil")
		}
	}
	return values
}
