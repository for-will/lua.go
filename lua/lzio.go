package golua

const EOZ = -1 /* end of stream */

type ZIO struct {
	n      int         /* bytes still unread */
	p      int         /* current position in buffer */
	reader LuaReadFunc //
	data   interface{} /* additional data*/
	L      *LuaState   /* Lua state (for reader) */
	buff   []byte
}

// Fill
// 对应C函数：`int luaZ_fill (ZIO *z)'
func (z *ZIO) Fill() int {
	z.L.Lock()
	buff, size := z.reader(z.L, z.data)
	z.L.Unlock()
	if buff == nil || size == 0 {
		return EOZ
	}
	z.n = size - 1
	z.p = 0
	z.p++
	return int(z.buff[z.p])
}

// Lookahead
// 对应C函数：`int luaZ_lookahead (ZIO *z)'
func (z *ZIO) Lookahead() int {
	if z.n == 0 {
		if z.Fill() == EOZ {
			return EOZ
		}
		/* luaZ_fill removed first byte; put back it */
		z.n++
		z.p--
	}
	return int(z.buff[z.p])
}

// Init
// 对应C函数：`void luaZ_init (lua_State *L, ZIO *z, lua_Reader reader, void *data)'
func (z *ZIO) Init(L *LuaState, reader LuaReadFunc, data interface{}) {
	z.L = L
	z.reader = reader
	z.data = data
	z.n = 0
	z.p = 0
}

// Read
// 对应C函数：`size_t luaZ_read (ZIO *z, void *b, size_t n)'
func (z *ZIO) Read(b []byte, n int) int {
	var pos = 0
	for n > 0 {
		if z.Lookahead() == EOZ {
			return n /*return number of missing bytes */
		}
		m := z.n
		if n <= m {
			m = n
		}
		copy(b[pos:], z.buff[z.p:z.p+m])
		z.n -= m
		z.p += m
		pos += m
		n -= m
	}
	return 0
}

// GetCh
// 对应C函数：`zgetc(z)'
func (z *ZIO) GetCh() int {
	var n = z.n
	z.n--
	if n > 0 {
		var c = z.buff[z.p]
		z.p++
		return int(c)
	} else {
		return z.Fill()
	}
}

// MBuffer
// 对应C结构体：`struct Mbuffer'
type MBuffer struct {
	buffer []byte
	n      int
	size   int
}

// Init
// 对应C函数：`luaZ_initbuffer(L, buff)'
func (m *MBuffer) Init() {
	m.buffer = nil
	m.size = 0
}

// OpenSpace
// 对应C函数：`char *luaZ_openspace (lua_State *L, Mbuffer *buff, size_t n)'
func (m *MBuffer) OpenSpace(n int) []byte {
	if n > m.size {
		if n < LUA_MINBUFFER {
			n = LUA_MINBUFFER
		}
		m.Resize(n)
	}
	return m.buffer
}

// Resize
// 对应C函数：`luaZ_resizebuffer'
func (m *MBuffer) Resize(n int) {
	b := make([]byte, n)
	copy(b, m.buffer)
	m.buffer = b
}

// Free
// 对应C函数：`luaZ_freebuffer(L, buff)'
func (m *MBuffer) Free() {
	m.Resize(0)
}

// Reset
// 对应C函数：`luaZ_resetbuffer(buff)'
func (m *MBuffer) Reset() {
	m.n = 0
}

// Len
// 对应C函数：`luaZ_bufflen(buff)'
func (m *MBuffer) Len() int {
	return m.n
}

func (m *MBuffer) string() string {
	return string(m.buffer[:m.n])
}
