package user

import (
	_ "github.com/proullon/ramsql/driver"
	"github.com/sean9999/transactor/services/address"
)

type User struct {
	ID        int
	Name      string
	Email     string
	Addresses []address.Address
}
