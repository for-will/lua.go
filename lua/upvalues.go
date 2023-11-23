package golua

/* UpValues */

type UpVal struct {
	CommonHeader
	v     *TValue /* points to stack or to its own value */
	value TValue  /* the value (when closed) */
	/* double linked list (when open) */
	lPrev *UpVal
	lNext *UpVal
}
