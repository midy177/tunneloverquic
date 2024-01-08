package tunneloverquic

type connection struct {
	connID int64
}

func newConnection() *connection {
	c := &connection{}
	return c
}
