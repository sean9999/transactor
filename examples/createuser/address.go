package main

import (
	"context"
	"database/sql"
	"errors"
	"iter"
	"time"

	"github.com/sean9999/transactor"
)

type Address struct {
	Street string
	Lat    float64
	Long   float64
	UserId int
	Id     int
}

var _ transactor.Op = (*CreateAddrsOp)(nil)

// CreateAddrsOp represents the operation to create an Addr.
type CreateAddrsOp struct {
	Addr *Address
	db   *sql.DB
	tx   *sql.Tx
}

func (c *CreateAddrsOp) Initialize(args ...any) error {
	return nil
}

func getLatLong(streetAddress string) (float64, float64, error) {
	time.Sleep(time.Second)
	if streetAddress == "INVALID ADDRESS" {
		return 0, 0, errors.New("invalid Addr")
	}
	return 45.612499, -73.707092, nil
}

func (c *CreateAddrsOp) Children() iter.Seq[transactor.Op] {
	return nil
}

func (c *CreateAddrsOp) Prepare(_ context.Context) error {

	lat, long, err := getLatLong(c.Addr.Street)
	if err != nil {
		return err
	}

	c.Addr.Lat = lat
	c.Addr.Long = long

	return nil
}

func (c *CreateAddrsOp) Commit(_ context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (c *CreateAddrsOp) Rollback() error {
	//TODO implement me
	panic("implement me")
}
