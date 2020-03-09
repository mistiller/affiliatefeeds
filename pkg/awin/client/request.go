package awinclient

type Request interface {
	URL() string
	Send() (result []byte, err error)
}
