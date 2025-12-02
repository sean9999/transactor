package address

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sean9999/transactor/transactor"
)

type Address struct {
	Street string
	Lat    float64
	Long   float64
	UserId int
	Id     int
}

var _ transactor.Op = (*CreateAddrOp)(nil)

// CreateAddrOp represents the operation to create an Addr.
type CreateAddrOp struct {
	Addr *Address
	db   *sql.DB
	tx   *sql.Tx
}

func getLatLong(streetAddress string) (float64, float64, error) {
	time.Sleep(time.Second)
	if streetAddress == "INVALID ADDRESS" {
		return 0, 0, errors.New("invalid Addr")
	}
	return 45.612499, -73.707092, nil
}

func (c *CreateAddrOp) Children() []transactor.Op {
	return nil
}

func (c *CreateAddrOp) Prepare(_ context.Context) error {

	lat, long, err := getLatLong(c.Addr.Street)
	if err != nil {
		return err
	}

	c.Addr.Lat = lat
	c.Addr.Long = long

	return nil
}

func (c *CreateAddrOp) Commit(_ context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (c *CreateAddrOp) Rollback() error {
	//TODO implement me
	panic("implement me")
}

//func (a *CreateAddrOp) prepare(ctx context.Context) error {
//	if a == nil {
//		return errors.New("Addr transactor is nil")
//	}
//	if ctx.Value("pleaseFail").(bool) == true {
//		return fmt.Errorf("operation failed")
//	}
//
//	//	fetch LatLong/long
//	a.Addr.LatLong = rand.Float64()
//	a.Addr.long = rand.Float64()
//
//	return nil
//}
