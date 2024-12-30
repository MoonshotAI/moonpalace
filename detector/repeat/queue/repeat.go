// https://www.codechef.com/viewsolution/28059224
// https://kimi.moonshot.cn/chat/ctp0o1kriic3gjf7ad2g

package repeat

import (
	"bufio"
	"fmt"
	"os"
)

const (
	mod = 1e9 + 7
	N   = 1e6 + 10
	inf = 1e9
)

// Ukkonen's Online Construction of Suffix Tree
type SuffixTree struct {
	pool            [N * 2]node
	rt              *node
	ptr             *node
	s               []int
	n               int
	uniqueSubstring int64
	leaf            []*node
	last            position
}

type node struct {
	goMap [26]*node
	link  *node
	fa    *node
	cnt   int
	st    int
	len   int
}

type position struct {
	u   *node
	cur int
	pos int
	rem int
}

func (ps *position) going(o int) {
	//fmt.Println("go ", o)
	ps.u = ps.u.goMap[o]
	ps.pos -= ps.u.len
	ps.cur += ps.u.len
}

func (ps *position) can(o int) bool {
	//fmt.Println("can ", o)
	if ps.u.goMap[o] != nil {
		//fmt.Println("can ", o)
		return ps.pos >= ps.u.goMap[o].len
	}
	return false
}

func (n *node) clr() {
	n.st = 0
	n.cnt = 0
	n.len = inf
	n.link = nil
	n.fa = nil
	for i := range n.goMap {
		n.goMap[i] = nil
	}
}

func (st *SuffixTree) newNode(fa *node, st1, len int) *node {
	var rtn node
	//rtn.clr()
	rtn.fa = fa
	rtn.st = st1
	rtn.len = len
	if fa != nil {
		fa.cnt++
	}
	return &rtn
}

func (st *SuffixTree) init() {
	st.n = 0
	st.ptr = &st.pool[0]
	st.rt = st.newNode(nil, 0, inf)
	st.last.u = st.rt
	st.last.cur = 0
	st.last.pos = 0
	st.last.rem = 0
	st.uniqueSubstring = 0
}

func (st *SuffixTree) walk() {
	for len(st.s) > st.last.cur && st.last.can(st.s[st.last.cur]) {
		//fmt.Println("walk")
		//fmt.Println(st.last.pos)
		st.last.going(st.s[st.last.cur])
	}
}

func (st *SuffixTree) followLink() {
	if st.last.u == st.rt {
		//fmt.Println("exe")
		if st.last.pos > 0 {
			//	fmt.Println("upd")
			st.last.pos--
			st.last.cur++
		}
	} else {
		st.last.u = st.last.u.link
		if st.last.u == nil {
			st.last.u = st.rt
		}
	}
	st.last.rem--
}

func (st *SuffixTree) add(c int) {
	//fmt.Println(c)
	st.s = append(st.s, c)
	st.n++
	st.last.rem++
	assert(st.last.pos >= 0)
	for p := st.rt; st.last.rem > 0; {
		if st.last.pos == 0 {
			st.last.cur = st.n - 1
		}
		st.walk()
		//fmt.Println("cur:", st.last.cur)
		o := st.s[st.last.cur]
		//fmt.Println("k ", o)
		t := st.s[0]
		v := &(st.last.u.goMap[o])
		//fmt.Println(st.last.u.goMap[0])
		if *v == nil {
			assert(st.last.pos == 0)
			//fmt.Println(st.last.pos)
			//fmt.Println("round1")
			vt := st.newNode(st.last.u, st.n-1, inf)
			//fmt.Println(st.last.u.goMap[0])
			st.last.u.goMap[o] = vt
			//afmt.Println(st.last.u.goMap[0])
			st.leaf = append(st.leaf, vt)
			(*p).link = st.last.u
			p = st.rt

		} else {
			t = st.s[(*v).st+st.last.pos]
			if t == c {
				//	fmt.Println("round2")
				(*p).link = st.last.u
				st.last.pos++
				st.uniqueSubstring += int64(len(st.leaf))
				return
			} else {
				//	fmt.Println("round3")
				u := st.newNode(st.last.u, (*v).st, st.last.pos)
				st.last.u.cnt--
				u.goMap[c] = st.newNode(u, st.n-1, inf)
				st.leaf = append(st.leaf, u.goMap[c])
				u.goMap[t] = *v
				u.cnt++
				(*v).fa = u
				(*v).st += st.last.pos
				(*v).len -= st.last.pos
				*v = u
				(*p).link = u
				p = u
			}
		}
		st.followLink()
	}
	st.uniqueSubstring += int64(len(st.leaf))
}

func (st *SuffixTree) del() {
	//	fmt.Println("delete")
	if st.last.pos > 0 {
		st.walk()
	}
	x := st.leaf[0]
	st.leaf = st.leaf[1:]
	for x != st.last.u && x.cnt == 0 {
		st.uniqueSubstring -= int64(min(st.n-x.st, x.len))
		p := x.fa
		p.goMap[st.s[x.st]] = nil
		p.cnt--
		x = x.fa
	}
	if st.last.rem > 0 && x == st.last.u {
		if st.last.pos == 0 && x.cnt == 0 {
			st.uniqueSubstring -= int64(min(st.n-x.st, x.len))
			if x.len != inf {
				x.st = st.n - x.len
				x.len = inf
			}
			st.uniqueSubstring += int64(min(st.n-x.st, x.len))
			st.leaf = append(st.leaf, x)
		} else if st.last.cur < st.n && x.goMap[st.s[st.last.cur]] == nil {
			u := st.newNode(x, st.n-st.last.pos, inf)
			x.goMap[st.s[st.last.cur]] = u
			st.uniqueSubstring += int64(min(st.n-u.st, u.len))
			st.leaf = append(st.leaf, u)
		} else {
			return
		}
		st.followLink()
	}
}

/*
func (st *SuffixTree) print(o *node) {
	if o.link != nil:

	fmt.Printf("idx=%d st=%d len=%d suf=%d\n", o-st.pool, o.st, o.len, o.link-st.pool)
	for i := range o.goMap {
		if o.goMap[i] != nil {
			assert(o.goMap[i].fa == o)
			fmt.Printf("%d -> %d\n", o-st.pool, o.goMap[i]-st.pool)
			st.print(o.goMap[i])
		}
	}
}
*/

func (st *SuffixTree) solve() {
	st.init()
	var q int
	fmt.Scan(&q)
	ret := int64(0)
	for i := 0; i < q; i++ {
		//fmt.Print("start")
		var op rune
		op = ' '
		for op != '+' && op != '-' {
			fmt.Scanf("%c", &op)
		}
		//fmt.Printf("%#U", op)
		if op == '+' {
			var c rune
			c = ' '
			for c < 'a' || 'z' < c {
				fmt.Scanf("%c", &c)
			}
			//fmt.Printf("%#U", c)
			st.add(int(c - 'a'))
		} else {
			st.del()
		}
		//fmt.Println("add")
		//st.uniqueSubstring %= mod
		//if st.uniqueSubstring < 0 {
		//	st.uniqueSubstring += mod
		//}
		ret += st.uniqueSubstring % mod
		////if ret < 0 {
		//	ret += mod
		//}
	}
	fmt.Printf("%d\n", int(ret%mod))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func assert(condition bool) {
	if !condition {
		panic("Assertion failed")
	}
}

// https://judge.yosupo.jp/problem/number_of_substrings
func yosupo() {
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	var s string
	fmt.Fscan(in, &s)
	suffixTree := &SuffixTree{}
	suffixTree.init()

	for _, c := range s {
		suffixTree.add(int(c - 'a'))
	}
	fmt.Fprintln(out, suffixTree.uniqueSubstring)
}

// https://www.codechef.com/problems/TMP01
func CodeChefTMP01() {
	suffixTree := &SuffixTree{}
	suffixTree.solve()
}

func main() {
	CodeChefTMP01()
	//yosupo()
}

// 本质不同的子串个数.
func (st *SuffixTree) CountSubstring() int64 {
	return st.uniqueSubstring
}

func (st *SuffixTree) GetRepeatness() float64 {
	z := st.uniqueSubstring
	n := st.n - st.leaf[0].st
	return float64(z) / float64(n*(n+1)/2)
}

func (st *SuffixTree) Length() int {
	return st.n - st.leaf[0].st
}
