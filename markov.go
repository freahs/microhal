package microhal

import (
	"fmt"
	"math/rand"
	"time"
)

var stopRunes []rune

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	stopRunes = []rune("!?.")
}

type markov struct {
	Order      int
	LeftChain  chain
	RightChain chain
}

func newMarkov(order int) *markov {
	return &markov{order, make(chain), make(chain)}
}

func (m *markov) GetKeywords(str string) (keywords []string) {
	runes := []rune(str)
	keywords = make([]string, len(runes)-m.Order+1)
	prefix := make(prefix, m.Order)
	for i := 0; i < len(runes); i++ {
		if i >= m.Order {
			keywords[i-m.Order] = prefix.String()
		}
		prefix.Shift(runes[i])
	}
	keywords[len(keywords)-1] = prefix.String()
	return
}

func (m *markov) AddString(str string) {
	runes := []rune(str)
	lPrefix, rPrefix := make(prefix, m.Order), make(prefix, m.Order)
	for r, l := 0, len(runes)-1; r < len(runes); r, l = r+1, l-1 {
		if r >= m.Order {
			m.LeftChain.Add(lPrefix.String(), runes[l])
			m.RightChain.Add(rPrefix.String(), runes[r])
		}
		lPrefix.Shift(runes[l])
		rPrefix.Shift(runes[r])
	}
}

func (m *markov) GetString(keyword string, maxLength int) (string, error) {
	if _, err := m.RightChain.Generate(keyword); err != nil {
		return "", err
	}

	lSlice, rSlice := make([]rune, maxLength), make([]rune, maxLength)
	lPrefix, rPrefix := make(prefix, m.Order), make(prefix, m.Order)

	kwdRunes := []rune(keyword)
	for r, l := 0, len(kwdRunes)-1; r < len(kwdRunes); r, l = r+1, l-1 {
		lPrefix.Shift(kwdRunes[l])
		rPrefix.Shift(kwdRunes[r])
	}

	i := 0
	for ; i < len(rSlice); i++ {
		r, err := m.RightChain.Generate(rPrefix.String())
		if err != nil {
			break
		}
		rSlice[i] = r
		if isStopRune(r) {
			break
		}
		rPrefix.Shift(r)
	}

	j := len(lSlice) - 1
	for ; j >= 0; j-- {
		r, err := m.LeftChain.Generate(lPrefix.String())
		if err != nil || isStopRune(r) {
			break
		}
		lSlice[j] = r
		lPrefix.Shift(r)
	}
	return string(lSlice[j+1:len(lSlice)]) + keyword + string(rSlice[:i]), nil
}

type chain map[string]*suffix

func (c chain) Add(s string, r rune) {
	if su, ok := c[s]; ok {
		su.Add(r)
	} else {
		su := &suffix{0, make(map[string]int)}
		su.Add(r)
		c[s] = su
	}
}

func (c chain) Generate(s string) (rune, error) {
	if su, ok := c[s]; ok {
		return su.Generate(), nil
	} else {
		return -1, fmt.Errorf("No such prefix.")
	}
}

type prefix []rune

func (p prefix) String() string {
	return string(p)
}

func (p prefix) Shift(r rune) {
	copy(p, p[1:])
	p[len(p)-1] = r
}

type suffix struct {
	N int
	M map[string]int
}

func (su *suffix) Add(r rune) {
	s := string(r)
	if val, ok := su.M[s]; ok {
		su.M[s] = val + 1
	} else {
		su.M[s] = 1
	}
	su.N++
}

func (su *suffix) Generate() rune {
	random := rand.Intn(su.N)
	i := 0
	for s, w := range su.M {
		i += w
		if random < i {
			return []rune(s)[0]
		}
	}
	panic("Internal error when generating suffix.")
}

func isStopRune(a rune) bool {
	for _, b := range stopRunes {
		if a == b {
			return true
		}
	}
	return false
}

func reverse(runes []rune) (rev []rune) {
	copy(rev, runes)
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return
}
