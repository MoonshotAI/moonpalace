// https://judge.yosupo.jp/problem/number_of_substrings
// Author: lychees

package repeat

import "sync"

var mapPool = sync.Pool{
	New: func() any {
		return make(map[rune]int32)
	},
}

func getMap() map[rune]int32 {
	return mapPool.Get().(map[rune]int32)
}

func putMap(x map[rune]int32) {
	for key := range x {
		delete(x, key)
	}
	mapPool.Put(x)
}

func newNode(link, maxLength int32) *node {
	return &node{
		Next:   getMap(),
		Link:   link,
		MaxLen: maxLength,
	}
}

type node struct {
	Next   map[rune]int32 // 孩子节点
	Link   int32          // 后缀链接
	MaxLen int32          // 当前节点对应的最长子串的长度
}

func NewSuffixAutomaton() *SuffixAutomaton {
	return &SuffixAutomaton{
		Nodes: []*node{
			newNode(0, 0),
		},
	}
}

type SuffixAutomaton struct {
	Nodes           []*node
	LastPos         int32 // 当前插入的字符对应的节点(终止点)
	n               int32 // 当前字符串长度
	uniqueSubstring int   // 不同子串数
}

func (sam *SuffixAutomaton) Clear() {
	for _, n := range sam.Nodes {
		putMap(n.Next)
	}
}

func (sam *SuffixAutomaton) AddString(s string) {
	for _, r := range s {
		sam.Add(r)
	}
}

func (sam *SuffixAutomaton) Add(c rune) {
	u := sam.LastPos
	uu := int32(len(sam.Nodes))
	sam.Nodes = append(sam.Nodes, newNode(0, sam.Nodes[u].MaxLen+1))
	for u != 0 && sam.Nodes[u].Next[c] == 0 {
		sam.Nodes[u].Next[c] = uu
		u = sam.Nodes[u].Link
	}
	if u == 0 && sam.Nodes[u].Next[c] == 0 {
		sam.Nodes[u].Next[c] = uu
		sam.Nodes[uu].Link = 0
	} else {
		v := sam.Nodes[u].Next[c]
		if sam.Nodes[v].MaxLen == sam.Nodes[u].MaxLen+1 {
			sam.Nodes[uu].Link = v
		} else {
			vv := int32(len(sam.Nodes))
			sam.Nodes = append(sam.Nodes, newNode(sam.Nodes[v].Link, sam.Nodes[u].MaxLen+1))
			for k, v2 := range sam.Nodes[v].Next {
				sam.Nodes[vv].Next[k] = v2
			}
			sam.Nodes[v].Link = vv
			sam.Nodes[uu].Link = vv
			for u != 0 && sam.Nodes[u].Next[c] == v {
				sam.Nodes[u].Next[c] = vv
				u = sam.Nodes[u].Link
			}
			if u == 0 && sam.Nodes[u].Next[c] == v {
				sam.Nodes[u].Next[c] = vv
			}
		}
	}
	sam.n++
	sam.LastPos = uu
	sam.uniqueSubstring += int(sam.h(uu))
}

// h
// pos 位置对应的子串个数.
// 用最长串的长度减去最短串的长度即可得到以当前节点为结尾的子串个数.
// 最长串的长度记录在节点的 MaxLength 中,最短串的长度可以通过link对应的节点的 MaxLength 加 1 得到.
func (sam *SuffixAutomaton) h(pos int32) int32 {
	return sam.Nodes[pos].MaxLen - sam.Nodes[sam.Nodes[pos].Link].MaxLen
}

// CountSubString 本质不同的子串个数.
func (sam *SuffixAutomaton) CountSubString() int {
	return sam.uniqueSubstring
}

func (sam *SuffixAutomaton) GetRepeatness() float64 {
	z := sam.uniqueSubstring
	n := sam.n
	return float64(z) / float64(n*(n+1)/2)
}

func (sam *SuffixAutomaton) Length() int32 {
	return sam.n
}
