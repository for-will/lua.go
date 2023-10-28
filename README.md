# lua.go

使用go语言实现lua-5.1.4虚拟机


## 一些代码片段

### LoadVar(S,x)的实现

```golang
// 对应C函数：`LoadVar(S,x)'
func (S *loadState) LoadVar(x interface{}) {
	// 应该考虑使用泛型？？
	size := int(reflect.TypeOf(x).Elem().Size())
	var buf = make([]byte, size)
	S.LoadMem(buf, 1, size)
	var p = reflect.ValueOf(x).Pointer()
	for i := 0; i < size; i++ {
		*(*byte)(unsafe.Pointer(p + uintptr(i))) = buf[i]
	}
}
```