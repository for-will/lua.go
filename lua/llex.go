package golua

import (
	"math"
	"strings"
	"unicode"
)

const (
	FIRST_RESERVED = 257

	/* maximum length of a reserved word */
	TOKEN_LEN = len("function")
)

type tk int

const (
	/* terminal symbols denoted by reserved words */
	TK_AND tk = iota + FIRST_RESERVED
	TK_BREAK
	TK_DO
	TK_ELSE
	TK_ELSEIF
	TK_END
	TK_FALSE
	TK_FOR
	TK_FUNCTION
	TK_IF
	TK_IN
	TK_LOCAL
	TK_NIL
	TK_NOT
	TK_OR
	TK_REPEAT
	TK_RETURN
	TK_THEN
	TK_TRUE
	TK_UNTIL
	TK_WHILE

	/* other terminal symbols */
	TK_CONCAT
	TK_DOTS
	TK_EQ
	TK_GE
	TK_LE
	TK_NE
	TK_NUMBER
	TK_NAME
	TK_STRING
	TK_EOS
)

const NUM_RESERVED = int(TK_WHILE - FIRST_RESERVED + 1)

/* ORDER RESERVED */
var luaXTokens = []string{
	"and", "break", "do", "else", "elseif",
	"end", "false", "for", "function", "if",
	"in", "local", "nil", "not", "or", "repeat",
	"return", "then", "true", "until", "while",
	"..", "...", "==", ">=", "<=", "~=",
	"<number>", "<name>", "<string>", "<eof>",
}

// 对应C函数：`void luaX_init (lua_State *L)'
func (L *LuaState) xInit() {
	for i := 0; i < NUM_RESERVED; i++ {
		ts := L.sNew([]byte(luaXTokens[i]))
		ts.Fix() /* reserved words are never collected */
		LuaAssert(len(luaXTokens[i]) <= TOKEN_LEN)
		ts.Reserved = lu_byte(i + 1) /* reserved word */
	}
}

// SemInfo
// 对应C结构：`union SemInfo'
type SemInfo struct {
	r  LuaNumber
	ts *TString
} /* semantics information */

// Token
// 对应C结构：`struct Token'
type Token struct {
	token   tk
	semInfo SemInfo
}

// LexState
// 对应C结构：`struct LexState'
type LexState struct {
	current    tk         /* current character (charint) */
	lineNumber int        /* input line counter */
	lastLine   int        /* line of last token `consumed' */
	t          Token      /* current token */
	lookAhead  Token      /* look ahead token */
	fs         *FuncState /* `FuncState' is private to the parser */
	L          *LuaState  /* */
	z          *ZIO       /* input stream */
	buff       *MBuffer   /* buffer for tokens */
	source     *TString   /* current source name */
	decPoint   byte       /* locale decimal point */
}

func xSetInput(L *LuaState, ls *LexState, z *ZIO, source *TString) {
	ls.decPoint = '.'
	ls.L = L
	ls.lookAhead.token = TK_EOS /* no look-ahead token */
	ls.z = z
	ls.fs = nil
	ls.lineNumber = 1
	ls.lastLine = 1
	ls.source = source
	ls.buff.Resize(LUA_MINBUFFER) /* initialize buffer */
	ls.next()                     /* read first char */
}

// 对应C函数：`next(ls)'
func (ls *LexState) next() {
	ls.current = tk(ls.z.GetCh())
}

// 对应C函数：`void luaX_next (LexState *ls)'
func (ls *LexState) xNext() {
	ls.lastLine = ls.lineNumber
	if ls.lookAhead.token != TK_EOS { /* is there a look-ahead token? */
		ls.t = ls.lookAhead         /* use this one */
		ls.lookAhead.token = TK_EOS /* and discharge it */
	} else {
		ls.t.token = ls.llex(&ls.t.semInfo) /* read next token */
	}
}

// 对应C函数：`static int llex (LexState *ls, SemInfo *seminfo)'
func (ls *LexState) llex(seminfo *SemInfo) tk {
	ls.buff.Reset()
	for {
		switch ls.current {
		case '\n', '\r':
			ls.incLineNumber()
			continue
		case '-':
			ls.next()
			if ls.current != '-' {
				return '-'
			}
			/* else is a comment */
			ls.next()
			if ls.current == '[' {
				var sep = ls.skipSep()
				ls.buff.Reset() /* `skip_sep' may dirty the buffer */
				if sep >= 0 {
					ls.readLongString(nil, sep) /* long comment */
					ls.buff.Reset()
					continue
				}
			}
			/* else short comment */
			for !ls.currIsNewline() && ls.current != EOZ {
				ls.next()
			}
			continue
		case '[':
			var sep = ls.skipSep()
			if sep >= 0 {
				ls.readLongString(seminfo, sep)
				return TK_STRING
			} else if sep == -1 {
				return '['
			} else {
				ls.xLexError("invalid long string delimiter", TK_STRING)
			}
		case '=':
			ls.next()
			if ls.current != '=' {
				return '='
			} else {
				ls.next()
				return TK_EQ
			}
		case '<':
			ls.next()
			if ls.current != '=' {
				return '<'
			} else {
				ls.next()
				return TK_LE
			}
		case '>':
			ls.next()
			if ls.current != '=' {
				return '>'
			} else {
				ls.next()
				return TK_GE
			}
		case '~':
			ls.next()
			if ls.current != '=' {
				return '~'
			} else {
				ls.next()
				return TK_NE
			}
		case '"', '\'':
			ls.readString(ls.current, seminfo)
			return TK_STRING
		case '.':
			ls.saveAndNext()
			if ls.checkNext(".") {
				if ls.checkNext(".") {
					return TK_DOTS /* ... */
				} else {
					return TK_CONCAT /* .. */
				}
			} else if !isdigit(ls.current) {
				return '.'
			} else {
				ls.readNumeral(seminfo)
				return TK_NUMBER
			}
		case EOZ:
			return TK_EOS
		default:
			if isspace(ls.current) {
				LuaAssert(!ls.currIsNewline())
				ls.next()
				continue
			} else if isdigit(ls.current) {
				ls.readNumeral(seminfo)
				return TK_NUMBER
			} else if isalpha(ls.current) || ls.current == '_' {
				/* identifier or reserved word */
				for isalnum(ls.current) || ls.current == '_' {
					ls.saveAndNext()
				}
				var ts = ls.xNewString(ls.buff.Bytes())
				if ts.Reserved > 0 { /* reserved word? */
					return tk(ts.Reserved-1) + FIRST_RESERVED
				} else {
					seminfo.ts = ts
					return TK_NAME
				}
			} else {
				var c = ls.current
				ls.next()
				return c /* single-char tokens (+ - / ...) */
			}
		}
	}
}

// 对应C函数：`static int check_next (LexState *ls, const char *set)'
func (ls *LexState) checkNext(set string) bool {
	if strings.IndexByte(set, byte(ls.current)) == -1 {
		return false
	}
	ls.saveAndNext()
	return true
}

/* LUA_NUMBER */
// 对应C函数：`static void read_numeral (LexState *ls, SemInfo *seminfo)'
func (ls *LexState) readNumeral(seminfo *SemInfo) {
	LuaAssert(isdigit(ls.current))
	for isdigit(ls.current) || ls.current == '.' {
		ls.saveAndNext()
	}
	if ls.checkNext("Ee") { /* `E'? */
		ls.checkNext("+-") /* optional exponent sign */
	}
	for isalnum(ls.current) || ls.current == '_' {
		ls.saveAndNext()
	}
	// ls.save(0)
	ls.buffReplace('.', ls.decPoint)           /* follow locale for decimal point */
	if !oStr2d(ls.buff.string(), &seminfo.r) { /* format error? */
		ls.tryDecPoint(seminfo) /* try to update decimal point separator */
	}
}

// 对应C函数：`static void buffreplace (LexState *ls, char from, char to)'
func (ls *LexState) buffReplace(from byte, to byte) {
	var n = ls.buff.Len()
	var p = ls.buff.buffer
	for i := 0; i < n; i++ {
		if p[i] == from {
			p[i] = to
		}
	}
}

// 对应C函数：`static void trydecpoint (LexState *ls, SemInfo *seminfo)'
func (ls *LexState) tryDecPoint(seminfo *SemInfo) {
	/* format error: try to update decimal point separator */
	var old = ls.decPoint
	if cv := localeconv(); cv != nil {
		ls.decPoint = cv.decimal_point[0]
	} else {
		ls.decPoint = '.'
	}
	ls.buffReplace(old, ls.decPoint) /* try update decimal separator */
	if !oStr2d(ls.buff.string(), &seminfo.r) {
		/* format error with correct decimal point: no more options */
		ls.buffReplace(ls.decPoint, '.') /* undo change (for error message) */
		ls.xLexError("malformed number", TK_NUMBER)
	}
}

// 对应C函数：`static void read_long_string (LexState *ls, SemInfo *seminfo, int sep)'
func (ls *LexState) readLongString(seminfo *SemInfo, sep int) {
	ls.saveAndNext()        /* skip 2nd `[' */
	if ls.currIsNewline() { /* string starts with a newline? */
		ls.incLineNumber() /* skip it */
	}
	var cont = 0
	for {
		switch ls.current {
		case EOZ:
			if seminfo != nil {
				ls.xLexError("unfinished long string", TK_EOS)
			} else {
				ls.xLexError("unfinished long comment", TK_EOS)
			}
		case '[': /* LUA_COMPAT_LSTR */
			if ls.skipSep() == sep {
				ls.saveAndNext() /* skip 2nd `[' */
			}
			cont++
			if LUA_COMPAT_LSTR == 1 && sep == 0 {
				ls.xLexError("nesting of [[...]] is deprecated", '[')
			}
		case ']':
			if ls.skipSep() == sep {
				ls.saveAndNext() /* skip 2nd `]' */
				if LUA_COMPAT_LSTR == 2 {
					cont--
					if sep == 0 && cont >= 0 {
						break
					}
				}
				goto endloop
			}
		case '\n', '\r':
			ls.save('\n')
			ls.incLineNumber()
			if seminfo == nil {
				ls.buff.Reset() /* avoid wasting space */
			}
		default:
			if seminfo != nil {
				ls.saveAndNext()
			} else {
				ls.next()
			}
		}
	}
endloop:
	if seminfo != nil {
		var buf = ls.buff
		seminfo.ts = ls.xNewString(buf.buffer[2+sep : buf.Len()-(2+sep)])
	}
}

// 对应C函数：`static void read_string (LexState *ls, int del, SemInfo *seminfo)'
func (ls *LexState) readString(del tk, seminfo *SemInfo) {
	ls.saveAndNext()
	for ls.current != del {
		switch ls.current {
		case EOZ:
			ls.xLexError("unfinished string", TK_EOS)
		case '\n', '\r':
			ls.xLexError("unfinished string", TK_STRING)
		case '\\':
			ls.next() /* do not save the '\' */
			var c tk
			switch ls.current {
			case 'a':
				c = '\a'
			case 'b':
				c = '\b'
			case 'f':
				c = '\f'
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			case 'v':
				c = '\v'
			case '\n', '\r':
				ls.save('\n')
				ls.incLineNumber()
				continue
			case EOZ:
				continue /* will raise an error next loop */
			default:
				if !isdigit(ls.current) {
					ls.saveAndNext() /* handles \\, \", \', and \? */
				} else { /* \xxx */
					c = 0
					for i := 0; i < 3 && isdigit(ls.current); i++ {
						c = 10*c + (ls.current - '0')
						ls.next()
					}
					if c > UCHAR_MAX {
						ls.xLexError("escape sequence too large", TK_STRING)
					}
					ls.save(c)
				}
				continue
			}
			ls.save(c)
			ls.next()
			continue
		default:
			ls.saveAndNext()
		}
	}
	ls.saveAndNext() /* skip delimiter */
	var buf = ls.buff
	seminfo.ts = ls.xNewString(buf.buffer[1 : buf.Len()-1])
}

// 对应C函数：`TString *luaX_newstring (LexState *ls, const char *str, size_t l)'
func (ls *LexState) xNewString(str []byte) *TString {
	var L = ls.L
	var ts = L.sNewStr(str)
	var o = ls.fs.h.SetByStr(L, ts) /* entry for `str' */
	if o.IsNil() {
		o.SetBoolean(true) /* make sure `str' will not be collected */
	}
	return ts
}

// 对应C函数：`static void inclinenumber (LexState *ls) '
func (ls *LexState) incLineNumber() {
	var old = ls.current
	LuaAssert(ls.currIsNewline())
	ls.next()
	if ls.currIsNewline() && ls.current != old {
		ls.next() /* skip '\n' or '\r' */
	}
	ls.lineNumber++
	if ls.lineNumber >= MAX_INT {
		ls.xSyntaxError("chunk has too many lines")
	}
}

// 对应C函数：`currIsNewline(ls)'
func (ls *LexState) currIsNewline() bool {
	return ls.current == '\n' || ls.current == '\r'
}

// 对应C函数：`void luaX_syntaxerror (LexState *ls, const char *msg)'
func (ls *LexState) xSyntaxError(msg string) {
	ls.xLexError(msg, ls.t.token)
}

// 对应C函数：`void luaX_lexerror (LexState *ls, const char *msg, int token)'
func (ls *LexState) xLexError(msg string, token tk) {
	const MAXSRC = 80
	var buff = make([]byte, 80)
	oChunkId(buff, string(ls.source.Bytes), MAXSRC)
	msg2 := ls.L.oPushFString("%s:%d: %s", buff, ls.lineNumber, msg)
	if token != 0 {
		ls.L.oPushFString("%s near "+LUA_QS, msg2, ls.txtToken(token))
	}
	ls.L.dThrow(LUA_ERRSYNTAX)
}

// 对应C函数：`static const char *txtToken (LexState *ls, int token)'
func (ls *LexState) txtToken(token tk) string {
	switch token {
	case TK_NAME, TK_STRING, TK_NUMBER:
		// ls.save(0)
		// return string(ls.buff.buffer)
		return ls.buff.string()
	default:
		return ls.xToken2str(token)
	}
}

// 对应C函数：`const char *luaX_token2str (LexState *ls, int token)'
func (ls *LexState) xToken2str(token tk) string {
	if token < FIRST_RESERVED {
		LuaAssert(token == tk(uint8(token)))
		if iscntrl(token) {
			return string(ls.L.oPushFString("char(%d)", token))
		} else {
			return string(ls.L.oPushFString("%c", byte(token)))
		}
	} else {
		return luaXTokens[token-FIRST_RESERVED]
	}
}

// 对应C函数：`static void save (LexState *ls, int c)'
func (ls *LexState) save(c tk) {
	var b = ls.buff
	if b.n+1 > b.size {
		if b.size > int(MAX_SIZET/2) {
			ls.xLexError("lexical element too long", 0)
		}
		var newSize = b.size * 2
		b.Resize(newSize)
	}
	b.buffer[b.n] = byte(c)
	b.n++
}

// 对应C函数：`save_and_next(ls)'
func (ls *LexState) saveAndNext() {
	ls.save(ls.current)
	ls.next()
}

// 对应C函数：`static int skip_sep (LexState *ls)'
func (ls *LexState) skipSep() int {
	var count = 0
	var s = ls.current
	LuaAssert(s == '[' || s == ']')
	ls.saveAndNext()
	for ls.current == '=' {
		ls.saveAndNext()
		count++
	}
	if ls.current == s {
		return count
	} else {
		return -count - 1
	}
}

// 对应C函数：`void luaX_lookahead (LexState *ls)'
func (ls *LexState) xLookAhead() {
	LuaAssert(ls.lookAhead.token == TK_EOS)
	ls.lookAhead.token = ls.llex(&ls.lookAhead.semInfo)
}

// ----------------------------------------------------------------------------

func iscntrl(c tk) bool {
	return unicode.IsControl(rune(c))
}

func isdigit(c tk) bool {
	return unicode.IsDigit(rune(c))
}

func isalpha(c tk) bool {
	return unicode.IsLetter(rune(c))
}

func isalnum(c tk) bool {
	var r = rune(c)
	return unicode.IsDigit(r) || unicode.IsLetter(r)
}

func isspace(c tk) bool {
	return unicode.IsSpace(rune(c))
}

func localeconv() *struct{ decimal_point string } {
	return nil
}

const UCHAR_MAX = math.MaxUint8
