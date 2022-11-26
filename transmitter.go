package tt

func CombineOut(h Handler, v ...Outer) Handler {
	if len(v) == 1 {
		return v[0].Out(h)
	}
	return CombineOut(h, v[1:]...)
}
