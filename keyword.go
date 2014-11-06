package microhal

import "sort"

const maxint = int(^uint(0) >> 1)

type _keyword struct {
	s string
	w int
}

type _kwSorter []_keyword

func (s _kwSorter) Len() int           { return len(s) }
func (s _kwSorter) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s _kwSorter) Less(i, j int) bool { return s[i].w < s[j].w }

func (s _kwSorter) Strings() (strings []string) {
	strings = make([]string, len(s))
	for i, k := range s {
		strings[i] = k.s
	}
	return
}

type keywords map[string]int

func (kw keywords) add(words []string) {
	for _, w := range words {
		if v, ok := kw[w]; ok {
			kw[w] = v + 1
		} else {
			kw[w] = 1
		}
	}
}

func (kw keywords) sort(words []string) (keywords []string) {
	sorter := make(_kwSorter, len(words))
	for i, s := range words {
		if w, ok := kw[s]; ok {
			sorter[i] = _keyword{s, w}
		} else {
			sorter[i] = _keyword{s, maxint}
		}
	}
	sort.Sort(sorter)
	keywords = sorter.Strings()
	return
}
