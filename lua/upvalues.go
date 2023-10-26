package golua

/* UpValues */

type UpVal struct {
	CommonHeader
	v     *TValue  /* points to stack or to its own value */
	value TValue   /* the value (when closed) */
	l     struct { /* double linked list (when open) */
		prev *UpVal
		next *UpVal
	}
}
