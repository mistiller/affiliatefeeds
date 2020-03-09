package storefront

type output interface {
	Dump() (dump []byte, err error)
}
