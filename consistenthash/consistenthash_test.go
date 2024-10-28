package consistenthash

import (
	"strconv"
	"testing"
)

func TestMap(t *testing.T) {
	m := New(3, func(data []byte) uint32 {
		i, _ := strconv.Atoi(string(data))
		return uint32(i)
	})

	testCase := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}
	m.Add("2", "4", "6")

	for k, v := range testCase {
		if mk := m.Get(k); mk != v {
			t.Errorf("expected real node %v, got %v", v, mk)
		}
	}
	testCase["27"] = "9"
	testCase["28"] = "9"
	m.Add("9")
	for k, v := range testCase {
		if mk := m.Get(k); mk != v {
			t.Errorf("after add expected real node %v, got %v", v, mk)
		}
	}
}
