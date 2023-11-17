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

## `OP_CALL`指令

OP_CALL对应的操作： `R(A), ... ,R(A+C-2) := R(A)(R(A+1), ... ,R(A+B-1))`

* `R(A)`是被调用的函数
* `R(A+1)`~`R(A+B-1)`是调用函数的参数
  * `B=1`表示函数有0个参数
  * `B=2`表示函数有1个参数
  * `B=3`表示函数有2个参数
  * 以此类推，函数个参数个数为`B-1`个
  * 总结：`B=0`表示B是一个无效值，所以要将参数个数+1，即`B=参数个数+1`
* `R(A)`~`R(A+C-2)`中存放函数的返回值
  * `C=1`表示没有返回值
  * `C=2`表示有1个返回值
  * `C=3`表示有2个返回值
  * 以此类推，函数返回值个数为`C-1`个
  * 总结：`C=0`表示C是一个无效值，所以要交返回值个数+1，即`C=返回值个数+1`