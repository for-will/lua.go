package golua

import (
	"luar/lua/mem"
	"math"
	"reflect"
)

const (
	MAXBITS  = 26
	MAXASIZE = 1 << MAXBITS /* max size of array part is 2^MAXBITS */
)

type TKey struct {
	TValue
	next *Node
}

func (k *TKey) GetTVal() *TValue {
	return &k.TValue
}

type Node struct {
	i_val TValue
	i_key TKey
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
	flags     lu_byte         // 1<<p means tagmethod(p) is not present
	lSizeNode lu_byte         // log2 of size of `node` array
	metatable *Table          /* */
	array     mem.Vec[TValue] /* */
	node      mem.Vec[Node]   /* */
	lastFree  int             // any free position is before this position
	gcList    GCObject        /* */
	sizeArray int             // size of `array` array
}

// const NumInts = int(unsafe.Sizeof(LuaNumber(0)) / unsafe.Sizeof(int(0)))

var DummyNodes = [1]Node{{
	i_val: TValue{},
	i_key: TKey{},
}}

var DummyNode = &DummyNodes[0]

func (t *Table) SizeNode() int {
	return 1 << t.lSizeNode
}

func (t *Table) GetNode(i int) *Node {
	return &t.node[i]
}

func (t *Table) HashPow2(n uint64) *Node {
	i := LMod(n, uint64(t.SizeNode()))
	return t.GetNode(int(i))
}

// HashMod 计算hash
// for some types, it is better to avoid modulus by power of 2, as
// they tend to have many 2 factors.
// 对应C函数：`hashmod(t,n)'
func (t *Table) HashMod(n uint64) *Node {
	n %= (uint64(t.SizeNode()) - 1) | 1
	return t.GetNode(int(n))
}

// HashNum
// hash for lua_Numbers
// 对应C函数：`static Node *hashnum (const Table *t, lua_Number n)'
func (t *Table) HashNum(n LuaNumber) *Node {
	if n == 0 { /* avoid problems with -0 */
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
	if b {
		return t.HashPow2(1)
	} else {
		return t.HashPow2(0)
	}
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
	switch key.gcType() {
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

// returns the index for `key` if `key` is an appropriate key to live in
// the array part of the table, -1 otherwise.
// 对应C函数：`static int arrayindex (const TValue *key)'
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

// returns the index of a `key' for table traversals. First goes all
// elements in the array part, then elements in the hash part. The
// beginning of a traversal is signalled by -1.
// 对应C函数：`static int findindex (lua_State *L, Table *t, StkId key)'
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
			if oRawEqualObj(n.GetKeyVal(), key) ||
				(n.GetKey().gcType() == LUA_TDEADKEY &&
					key.IsCollectable() &&
					n.GetKey().GcValue() == key.GcValue()) {
				i = mem.ElemIndex(t.node, n)
				/* hash elements are numbered after array ones */
				return i + t.sizeArray
			}
			n = n.GetNext()
		}
		L.DbgRunError("invalid key to 'next'") /* key not found */
		return 0                               /* to avoid warnings */
	}
}

// hNext 对Table进行迭代, 在key处存放key值，在key+1处存放value值，如果没有元素返回false
// 同C函数 `int luaH_next (lua_State *L, Table *t, StkId key)`
func (t *Table) hNext(L *LuaState, key StkId) bool {
	i := t.findIndex(L, key) /* find original element */
	i++
	for ; i < t.sizeArray; i++ { /* try first array part */
		if !t.array[i].IsNil() { /* a non-nil value? */
			key.SetNumber(LuaNumber(i + 1))
			SetObj(L, key.Ptr(1), &t.array[i])
			return true
		}
	}
	for i -= t.sizeArray; i < t.SizeNode(); i++ { /* then hash part */
		if n := t.GetNode(i); !n.GetVal().IsNil() { /* a non-nil value? */
			key.SetObj(L, n.GetKeyVal())
			key.Ptr(1).SetObj(L, n.GetVal())
			return true
		}
	}
	return false /* no more elements */
}

/* Rehash */

// 返回：na - 数组中存放元素的数量 ； n - 新的数组的大小；
// 对应C函数 `static int computesizes (int nums[], int *narray)`
func computeSizes(nums []int, nArray int) (na, n int) {
	var twotoi = 1 /* 2^i */
	var a = 0      /* number of elements smaller than 2^i */
	na = 0         /* number of elements to go to array */
	n = 0          /* optimal size for array part */

	for i := 0; twotoi/2 < nArray; i++ {
		if nums[i] > 0 {
			a += nums[i]
			if a > twotoi/2 { /* more than half elements present? */
				n = twotoi /* optimal size (still now) */
				na = a     /* all elements smaller than n will goto array part */
			}
		}
		if a == nArray {
			break /* all elements already counted */
		}
		twotoi *= 2
	}
	return na, n
}

// 对应C函数 `static int countint (const TValue *key, int *nums)`
func countInt(key *TValue, nums []int) int {
	var k = arrayIndex(key)
	if 0 < k && k <= MAXASIZE { /* is `key` an appropriate array index? */
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
func (t *Table) setArrayVector(size int, h mem.ErrorHandler) {
	t.array.ReAlloc(size, h)
	for i := t.sizeArray; i < size; i++ {
		t.array[i].SetNil()
	}
	t.sizeArray = size
}

// 对应C函数 `static void setnodevector (lua_State *L, Table *t, int size)`
func (t *Table) setNodeVector(L *LuaState, size int) {

	var lSize int

	// no elements to hash part?
	if size == 0 {
		t.node = DummyNodes[:]
		lSize = 0
	} else {
		lSize = CeilLog2(uint64(size))
		if lSize > MAXBITS {
			L.DbgRunError("table overflow")
		}
		size = 1 << lSize
		t.node.Init(size, L)
		for i := 0; i < size; i++ {
			n := t.GetNode(i)
			n.SetNext(nil)
			n.GetKeyVal().SetNil()
			n.GetVal().SetNil()
		}
	}
	t.lSizeNode = lu_byte(lSize)
	t.lastFree = size // all positions are free
}

// resize 重新构建表，新构建表的数组部分大小为nasize，散列哈希部分的大小为nhsize
// 对应C函数：`static void resize (lua_State *L, Table *t, int nasize, int nhsize)'
func (t *Table) resize(L *LuaState, nasize int, nhsize int) {
	var (
		oldArraySize = t.sizeArray
		oldHashSize  = t.lSizeNode
		oldNodes     = t.node // save old hash
	)
	if nasize > oldArraySize { /* array part must grow? */
		t.setArrayVector(nasize, L)
	}
	/* create new hash part with appropriate size */
	t.setNodeVector(L, nhsize)
	if nasize < oldArraySize { /* array part must shrink? */
		t.sizeArray = nasize
		/* re-insert elements from vanishing slice */
		for i := nasize; i < oldArraySize; i++ {
			if !t.array[i].IsNil() {
				v := t.SetByNum(L, i+1)
				SetObj(L, v, &t.array[i])
			}
		}
		/* shrink array */
		t.array.ReAlloc(nasize, L)
	}
	/* re-insert elements from hash part */
	for i := 1<<oldHashSize - 1; i >= 0; i-- {
		var old = &oldNodes[i]
		if !old.GetVal().IsNil() {
			v := t.Set(L, old.GetKeyVal())
			SetObj(L, v, old.GetVal())
		}
	}

	/* gc回收oldNodes的内存，不用作其他处理 */
	// oldNodes.Free(L)
}

// ResizeArray 重新分配数组部分的大小
// 对应C函数 `void luaH_resizearray (lua_State *L, Table *t, int nasize)`
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

// hNew 创建一个数组部分长度为narray，散列部分长度为nhash的Table,
// 并返回新创建Table的指针。
// 同C函数 `Table *luaH_new (lua_State *L, int narray, int nhash)`
func (L *LuaState) hNew(narray int, nhash int) *Table {
	t := &Table{
		metatable: nil,
		flags:     ^lu_byte(0),
		array:     nil,
		sizeArray: 0,
		lSizeNode: 0,
		node:      DummyNodes[:],
	}
	L.cLink(t, LUA_TTABLE)
	t.setArrayVector(narray, L)
	t.setNodeVector(L, nhash)
	return t
}

func (t *Table) getFreePos() *Node {
	for t.lastFree > 0 {
		t.lastFree--
		if n := t.GetNode(t.lastFree); n.GetKey().IsNil() {
			return n
		}
	}
	return nil /* could not find a free place */
}

// newKey 在table中添加插入新的key并返回对应value的指针
// 对应C函数 `static TValue *newkey (lua_State *L, Table *t, const TValue *key)`
// inserts a new key into a has table; first, check whether key's main
// position is free. If not, check whether colliding node is in its main
// position or not: if it is not, move colliding node to an empty place and
// put new key in its main position; otherwise (colliding node is in its main
// position), new key goes to an empty position.
// 对应C函数：`static TValue *newkey (lua_State *L, Table *t, const TValue *key)'
func (t *Table) newKey(L *LuaState, key *TValue) *TValue {
	mp := t.MainPosition(key)
	if !mp.GetVal().IsNil() || mp == DummyNode {
		n := t.getFreePos() // get a free place
		if n == nil {       // cannot find a free place?
			t.rehash(L, key)     // grow table
			return t.Set(L, key) // re-insert key into grown table
		}
		LuaAssert(n != DummyNode)
		on := t.MainPosition(mp.GetKeyVal())

		// is colliding node out of its main position?
		if on != mp {
			// yes; move colliding node into free position
			for on.GetNext() != mp {
				on = on.GetNext() // find previous
			}
			on.SetNext(n)   // redo the chain with `n` in place of `mp`
			*n = *mp        // copy colliding node into free pos. (mp->next also goes)
			mp.SetNext(nil) // now `mp` is free
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
	L.cBarrierT(t, key)
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

// GetByString search function for string
// 对应C函数 `const TValue *luaH_getstr (Table *t, TString *key)'
func (t *Table) GetByString(key *TString) *TValue {
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
	switch key.gcType() {
	case LUA_TNIL:
		return LuaObjNil
	case LUA_TSTRING:
		return t.GetByString(key.StringValue())
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
			if oRawEqualObj(n.GetKey().GetTVal(), key) {
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
			L.DbgRunError("table index is nil")
		} else if key.IsNumber() && math.IsNaN(key.NumberValue()) {
			L.DbgRunError("table index is NaN")
		}
		return t.newKey(L, key)
	}
}

// SetByNum 获取key在t中对应的TValue的指针，如果t中不存在则创建并返回
// 同C函数 `TValue *luaH_setnum (lua_State *L, Table *t, int key)`
func (t *Table) SetByNum(L *LuaState, key int) *TValue {
	p := t.GetNum(key)
	if p != LuaObjNil {
		return p
	} else {
		k := &TValue{}
		k.SetNumber(LuaNumber(key))
		return t.newKey(L, k)
	}
}

// SetByStr
// 同C函数：`TValue *luaH_setstr (lua_State *L, Table *t, TString *key)'
func (t *Table) SetByStr(L *LuaState, key *TString) *TValue {
	if p := t.GetByString(key); p != LuaObjNil {
		return p
	} else {
		var k TValue
		k.SetString(L, key)
		return t.newKey(L, &k)
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
