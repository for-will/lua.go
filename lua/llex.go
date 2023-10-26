package golua

const (
	FIRST_RESERVED = 257

	/* maximum length of a reserved word */
	TOKEN_LEN = len("function")
)

const (
	/* terminal symbols denoted by reserved words */
	TK_AND = iota + FIRST_RESERVED
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

const NUM_RESERVED = TK_WHILE - FIRST_RESERVED + 1

/* ORDER RESERVED */
var luaXTokens = []string{
	"and", "break", "do", "else", "elseif",
	"end", "false", "for", "function", "if",
	"in", "local", "nil", "not", "or", "repeat",
	"return", "then", "true", "until", "while",
	"..", "...", "==", ">=", "<=", "~=",
	"<number>", "<name>", "<string>", "<eof>",
}

func (L *LuaState) LuaXInit() {
	for i := 0; i < NUM_RESERVED; i++ {
		ts := L.sNew([]byte(luaXTokens[i]))
		ts.Fix() // reserved words are never collected
		LuaAssert(len(luaXTokens[i]) <= TOKEN_LEN)
		ts.Reserved = lu_byte(i + 1) // reserved word
	}
}
