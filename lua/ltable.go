package golua

import (
	"math"
	"reflect"
	"unsafe"
)

const MAXBITS = 26

type TKey struct {
	TValue
	next *Node
}

type Node struct {
	i_val TValue
	i_key TKey
}

func (n *Node) OffsetFrom(origin *Node) int {
	offset := uintptr(unsafe.Pointer(origin)) - uintptr(unsafe.Pointer(n))
	return int(offset / unsafe.Sizeof(*n))
}

func (n *Node) GetVal() *TValue {
	return &n.i_val
}

func (n *Node) GetKey() *TKey {
	return &n.i_key
}

// GetKeyVal 获取key的值
// 对应C `#define key2tval(n)	(&(n)->i_key.tvk)`
func (n *Node) GetKeyVal() *TValue {
	return &n.i_key.TValue
}

func (n *Node) GetNext() *Node {
	return n.i_key.next
}

func (n *Node) SetNext(next *Node) {
	n.i_key.next = next
}

type Table struct {
	CommonHeader
	flags     lu_byte // 1<<p means tagmethod(p) is not present
	lSizeNode lu_byte // log2 of size of `node` array
	metatable *Table
	array     []TValue
	node      []Node
	lastFree  int // any free position is before this position
	gcList    GCObject
	sizeArray int // size of `array` array
}

// const NumInts = int(unsafe.Sizeof(LuaNumber(0)) / unsafe.Sizeof(int(0)))

var DummyNodes = [1]Node{{
	i_val: TValue{},
	i_key: TKey{},
}}

var DummyNode = &DummyNodes[0]

func (t *Table) SizeNode() uint64 {
	return 1 << t.lSizeNode
}

func (t *Table) GetNode(i uint64) *Node {
	return &t.node[i]
}

func (t *Table) IndexNode(n *Node) int {
	return n.OffsetFrom(&t.node[0])
}

func (t *Table) HashPow2(n uint64) *Node {
	i := LMod(n, t.SizeNode())
	return t.GetNode(i)
}

// HashMod 计算hash
// for some types, it is better to avoid modulus by power of 2, as
// they tend to have many 2 factors.
func (t *Table) HashMod(n uint64) *Node {
	n %= (t.SizeNode() - 1) | 1
	return t.GetNode(n)
}

func (t *Table) HashNum(n LuaNumber) *Node {
	if n == 0 {
		// avoid problems with -0
		return t.GetNode(0)
	}
	// 在64位系统下uint和lua_Number(float64)都是64位，直接计算hash
	var a = math.Float64bits(n)
	return t.HashMod(a)
}

func (t *Table) HashStr(str *TString) *Node {
	return t.HashPow2(str.Hash)
}

func (t *Table) HashBoolean(b LuaBoolean) *Node {
	return t.HashPow2(uint64(b))
}

func (t *Table) HashPointer(p interface{}) *Node {
	// *(uint64 *)unsafe.Pointer(p)
	// todo: 考虑栈伸缩问题
	ptr := reflect.ValueOf(p).Pointer()
	return t.HashMod(uint64(ptr))
}

// MainPosition 获取key在表中的hash位置
// returns the `main` position of an element in a table (that is, the index
// of its hash value)
func (t *Table) MainPosition(key *TValue) *Node {
	switch key.Type() {
	case LUA_TNUMBER:
		return t.HashNum(key.NumberValue())
	case LUA_TSTRING:
		return t.HashStr(key.StringValue())
	case LUA_TBOOLEAN:
		return t.HashBoolean(key.BooleanValue())
	case LUA_TLIGHTUSERDATA:
		return t.HashPointer(key.PointerValue())
	default:
		return t.HashPointer(key.GcValue())
	}
}

// returns the index for `key` is `key` is an appropriate key to live in
// the array part of the table, -1 otherwise.
func arrayIndex(key *TValue) int {
	if key.IsNumber() {
		n := key.NumberValue()
		k := int(n)
		if LuaNumber(k) == n {
			return k
		}
	}
	return -1 /* `key` did not match some condition */
}

func (t *Table) findIndex(L *LuaState, key StkId) int {
	if key.IsNil() {
		return -1 /* first iteration */
	}
	i := arrayIndex(key)
	if 0 < i && i <= t.sizeArray { /* is `key` inside array part? */
		return i - 1 /* yes; that's the index (corrected to C) */
	} else {
		n := t.MainPosition(key)

		for n != nil { /* check whether `key` is somewhere in the chain */
			/* key may be dead already, but it is ok to use it in `next` */
			if n.GetVal().IsEqualTo(key) ||
				(n.GetKey().Type() == LUA_TDEADKEY &&
					key.IsCollectable() &&
					n.GetKey().GcValue() == key.GcValue()) {
				i = t.IndexNode(n)
				/* hash elements are numbered after array ones */
				return i + t.sizeArray
			}
			n = n.GetNext()
		}
		L.DebugRunError("invalid key to 'next'") /* key not found */
		return 0                                 /* to avoid warnings */
	}
}

// hNext 对Table进行迭代
// 同C函数 `int luaH_next (lua_State *L, Table *t, StkId key)`
func (t *Table) hNext(L *LuaState, key StkId) int {
	i := t.findIndex(L, key) /* find original element */
	i++
	for ; i < t.sizeArray; i++ { /* try first array part */
		if !t.array[i].IsNil() { /* a non-nil value? */
			key.SetNumber(LuaNumber(i + 1))
			SetObj(L, key.PtrAdd(1), &t.array[i])
			return 1
		}
	}
	for i -= t.sizeArray; i < int(t.SizeNode()); i++ { /* then hash part */
		if !t.GetNode(uint64(i)).GetVal().IsNil() { /* a non-nil value? */
			SetObj(L, key, t.GetNode(uint64(i)).GetKeyVal())
			SetObj(L, key.PtrAdd(1), t.GetNode(uint64(i)).GetVal())
			return 1
		}
	}
	return 0 /* no more elements */
}

/* Rehash */

// 返回：elements - 数组中存放元素的数量 ； size - 新的数组的大小；
// 对应C函数 `static int computesizes (int nums[], int *narray)`
func computeSizes(nums []int, narray int) (elements, size int) {
	var (
		i      = 0
		twotoi = 1 // 2^i
		a      = 0 // number of elements smaller than 2^i
		na     = 0 // number of elements to go to array
		n      = 0 // optimal size for array part
	)

	for twotoi/2 < narray {
		if nums[i] > 0 {
			a += nums[i]
			// more than half elements present?
			if a > twotoi/2 {
				n = twotoi // optimal size (still now)
				na = a     // all elements smaller than n will goto array part
			}
		}
		if a == narray {
			break // all elements already counted
		}
	}
	return na, n
}

// 对应C函数 `static int countint (const TValue *key, int *nums)`
func countInt(key *TValue, nums []int) int {
	k := arrayIndex(key)

	// is `key` an appropriate array index?
	if 0 < k && k <= MAXBITS {
		nums[CeilLog2(uint64(k))]++ // count as such
		return 1
	}
	return 0
}

// numUseArray 统计数组部分的数据分布到nums中，返回数组部分存储的数据总数
// 对应C函数 `static int numusearray (const Table *t, int *nums)`
func (t *Table) numUseArray(nums []int) int {
	var (
		lg   = 0
		ttlg = 1 // 2^lg
		ause = 0 // summation of `nums`
		i    = 1 // count to traverse all array keys
	)

	for lg <= MAXBITS { // or each slice
		var lc = 0 // counter
		var lim = ttlg
		if lim > t.sizeArray {
			lim = t.sizeArray // adjust upper limit
			if i > lim {
				break // no more elements to count
			}
		}
		/* count elements in rang (2^(lg-1), 2^lg] */
		for ; i <= lim; i++ {
			if !t.array[i-1].IsNil() {
				lc++
			}
		}
		nums[lg] += lc
		ause += lc
		lg++
		ttlg *= 2
	}
	return ause
}

func (t *Table) numUseHash(nums []int) (totaluse int, ause int) {
	var i = t.SizeNode()
	for i > 0 {
		i--
		n := t.GetNode(i)
		if !n.GetVal().IsNil() {
			ause += countInt(n.GetKeyVal(), nums)
			totaluse++
		}
	}
	return totaluse, ause
}

// 对应C函数 `static void setarrayvector (lua_State *L, Table *t, int size) `
func (t *Table) setArrayVector(size int) {
	// luaM_reallocvector(L, t->array, t->sizearray, size, TValue);

	// 这里直接重新申请，让go自己管理内存
	narray := make([]TValue, size)
	copy(narray, t.array)
	for i := t.sizeArray; i < size; i++ {
		t.array[i].SetNil()
	}
	t.array = narray
	t.sizeArray = size
}

// 对应C函数 `static void setnodevector (lua_State *L, Table *t, int size)`
func (t *Table) setNodeVector(L *LuaState, size int) {

	var lsize int

	// no elements to hash part?
	if size == 0 {
		t.node = DummyNodes[:]
		lsize = 0
	} else {
		lsize = CeilLog2(uint64(size))
		if lsize > MAXBITS {
			L.DebugRunError("table overflow")
		}
		size = 1 << size
		t.node = make([]Node, size)
		for i := 0; i < size; i++ {
			n := t.GetNode(uint64(i))
			n.SetNext(nil)
			n.GetKeyVal().SetNil()
			n.GetVal().SetNil()
		}
	}
	t.lSizeNode = lu_byte(lsize)
	t.lastFree = size - 1 // all positions are free
}

// resize 重新构建表，新构建表的数组部分大小为nasize，散列哈希部分的大小为nhsize
func (t *Table) resize(L *LuaState, nasize int, nhsize int) {
	var (
		oldasize = t.sizeArray
		oldhsize = t.lSizeNode
		nold     = t.node // save old hash
	)
	if nasize > oldasize { /* array part must grow? */
		t.setArrayVector(nasize)
	}
	/* create new hash part with appropriate size */
	t.setNodeVector(L, nhsize)
	if nasize < oldasize { /* array part must shrink? */
		t.sizeArray = nasize
		/* re-insert elements from vanishing slice */
		for i := nasize; i < oldasize; i++ {
			if !t.array[i].IsNil() {
				v := t.SetNum(L, i+1)
				SetObj(L, v, &t.array[i])
			}
		}
		/* shrink array */
		narray := make([]TValue, nasize)
		copy(narray, t.array)
		t.array = narray
	}
	/* re-insert elements from hash part */
	for i := 1<<oldhsize - 1; i >= 0; i-- {
		old := &nold[i]
		if !old.GetVal().IsNil() {
			v := t.Set(L, old.GetVal())
			SetObj(L, v, old.GetVal())
		}
	}

	/* gc回收nold的内存，不用作其他处理 */
}

// ResizeArray 重新分配数组部分的大小
// 同C函数 `void luaH_resizearray (lua_State *L, Table *t, int nasize)`
func (t *Table) ResizeArray(L *LuaState, nasize int) {
	var nsize = 0
	if &t.node[0] != DummyNode {
		nsize = int(t.SizeNode())
	}
	t.resize(L, nasize, nsize)
}

// rehash 做重新散列操作
// 对应C函数 `static void rehash (lua_State *L, Table *t, const TValue *ek)`
func (t *Table) rehash(L *LuaState, ek *TValue) {
	var (
		nasize   int // 数组总共可存放元素的数量
		na       int // 数组中已存放的元素的数量
		nums     [MAXBITS]int
		totaluse int
	)

	nasize = t.numUseArray(nums[:])               // count keys in array part
	totaluse = nasize                             // all those keys are integer keys
	hashTotal, hashArray := t.numUseHash(nums[:]) // count keys in hash part
	totaluse += hashTotal
	nasize += hashArray
	// count extra key
	nasize += countInt(ek, nums[:])
	totaluse++
	// compute new size for array part
	na, nasize = computeSizes(nums[:], nasize)
	// resize the table to new computed sizes
	t.resize(L, nasize, totaluse-na)
}

// NewTable 创建一个数组部分长度为narray，散列部分长度为nhash的Table,
// 并返回新创建Table的指针。
// 同C函数 `Table *luaH_new (lua_State *L, int narray, int nhash)`
func NewTable(L *LuaState, narray int, nhash int) *Table {
	t := &Table{
		metatable: nil,
		flags:     ^lu_byte(0),
		array:     nil,
		sizeArray: 0,
		lSizeNode: 0,
		node:      DummyNodes[:],
	}
	// todo: luaC_link(L, obj2gco(t), LUA_TTABLE)
	t.setArrayVector(narray)
	t.setNodeVector(L, nhash)
	return t
}

func (t *Table) getFreePos() *Node {
	for t.lastFree > 0 {
		t.lastFree--
		if n := t.GetNode(uint64(t.lastFree)); n.GetKey().IsNil() {
			return n
		}
	}
	return nil // could not find a free place
}

// NewKey 在table中添加插入新的key并返回对应value的指针
// 对应C函数 `static TValue *newkey (lua_State *L, Table *t, const TValue *key)`
// inserts a new key into a has table; first, check whether key's main
// position is free. If not, check whether colliding node is in its main
// position or not: if it is not, move colliding node to an empty place and
// put new key in its main position; otherwise (colliding node is in its main
// position), new key goes to an empty position.
func (t *Table) NewKey(L *LuaState, key *TValue) *TValue {
	mp := t.MainPosition(key)
	if !mp.GetVal().IsNil() || mp == DummyNode {
		n := t.getFreePos() // get a free place
		if n == nil {       // cannot find a free place?
			t.rehash(L, key)     // grow table
			return t.Set(L, key) // re-insert key into grown table
		}
		LuaAssert(n != DummyNode)
		othern := t.MainPosition(mp.GetVal())

		// is colliding node out of its main position?
		if othern != mp {
			// yes; move colliding node into free position
			for othern.GetNext() != mp {
				othern = othern.GetNext() // find previous
			}
			othern.SetNext(n) // redo the chain with `n` in place of `mp`
			*n = *mp          // copy colliding node into free pos. (mp->next also goes)
			mp.SetNext(nil)   // now `mp` is free
			mp.GetVal().SetNil()
		} else {
			// colliding node is in its own main position
			// new node will go into free position
			n.SetNext(mp.GetNext()) // chain new position
			mp.SetNext(n)
			mp = n
		}
	}
	mp.GetKey().TValue = *key
	// todo: luaC_barriert(L, t, key);
	LuaAssert(mp.GetVal().IsNil())
	return mp.GetVal()
}

// GetNum search function for integers
// 对应C函数 `const TValue *luaH_getnum (Table *t, int key)'
func (t *Table) GetNum(key int) *TValue {
	// (1 <= key && key <= t->sizeArray)
	if uint(key-1) < uint(t.sizeArray) {
		return &t.array[key-1]
	} else {
		nk := LuaNumber(key)
		n := t.HashNum(nk)
		for n != nil {
			// check whether `key` is somewhere in the chain
			if k := n.GetKey(); k.IsNumber() && k.NumberValue() == nk {
				return n.GetVal() // that's it
			} else {
				n = n.GetNext()
			}
		}
		return LuaObjNil
	}
}

// GetString search function for string
// 对应C函数 `const TValue *luaH_getstr (Table *t, TString *key)'
func (t *Table) GetString(key *TString) *TValue {
	n := t.HashStr(key)
	for n != nil {
		if n.GetKey().IsString() && n.GetKey().StringValue() == key {
			return n.GetVal()
		}
		n = n.GetNext()
	}
	return LuaObjNil
}

// Get main search function
// 对应C函数 `const TValue *luaH_get (Table *t, const TValue *key)`
func (t *Table) Get(key *TValue) *TValue {
	switch key.Type() {
	case LUA_TNIL:
		return LuaObjNil
	case LUA_TSTRING:
		return t.GetString(key.StringValue())
	case LUA_TNUMBER:
		n := key.NumberValue()
		k := int(n)
		if LuaNumber(k) == n { // index is int?
			return t.GetNum(k) // use specialized version
		}
		fallthrough // else go through
	default:
		n := t.MainPosition(key)
		for n != nil {
			// check whether `key` is somewhere in the chain
			if n.GetKey().IsEqualTo(key) {
				return n.GetVal() // that's it
			} else {
				n = n.GetNext()
			}
		}
		return LuaObjNil
	}
}

// Set 获取table中key对应的value，如果不存在则新创建
// 对应C函数 `TValue *luaH_set (lua_State *L, Table *t, const TValue *key)`
func (t *Table) Set(L *LuaState, key *TValue) *TValue {
	p := t.Get(key)
	t.flags = 0
	if p != LuaObjNil {
		return p
	} else {
		if key.IsNil() {
			// todo: luaG_runerror(L, "table index is nil")
		} else if key.IsNumber() && math.IsNaN(key.NumberValue()) {
			// todo: luaG_runerror(L, "table index is NaN");
		}
		return t.NewKey(L, key)
	}
}

// SetNum 获取key在t中对应的TValue的指针，如果t中不存在则创建并返回
// 同C函数 `TValue *luaH_setnum (lua_State *L, Table *t, int key)`
func (t *Table) SetNum(L *LuaState, key int) *TValue {
	p := t.GetNum(key)
	if p != LuaObjNil {
		return p
	} else {
		k := &TValue{}
		k.SetNumber(LuaNumber(key))
		return t.NewKey(L, k)
	}
}

// 同C函数 `static int unbound_search (Table *t, unsigned int j)'
func (t *Table) unboundSearch(j int) int {
	i := j /* i is zero or a present index */
	j++
	/* find `i' and `j' such that i is present and j is not */
	for !t.GetNum(j).IsNil() {
		i = j
		j *= 2
		if j > MAX_INT { /* overflow? */
			/* table was built with bad purposes: resort to linear search */
			i = 1
			for ; t.GetNum(i).IsNil(); i++ {
			}
			return i - 1
		}
	}
	/* now do a binary search between them */
	for j-i > 1 {
		m := (i + j) / 2
		if t.GetNum(m).IsNil() {
			j = m
		} else {
			i = m
		}
	}
	return i
}

// GetN
// Try to find a boundary in table `t'. A `boundary' is an integer index
// such that t[i] is non-nil and t[i+1] is nil (and 0 if t[1] is nil).
// 同C函数 `int luaH_getn (Table *t)'
func (t *Table) GetN() int {
	j := t.sizeArray
	if j > 0 && t.array[j-1].IsNil() {
		/* there is a boundary in the array part: (binary) search for it */
		var i = 0
		for j-i > 1 {
			m := (i + j) / 2
			if t.array[m-1].IsNil() {
				j = m
			} else {
				i = m
			}
		}
		return i
		/* else must find a boundary in hash part */
	} else if &t.node[0] == DummyNode { /* hash part is empty? */
		return j /* that is easy... */
	} else {
		return t.unboundSearch(j)
	}
}
