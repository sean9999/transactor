package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sean9999/transactor/services/address"
	"github.com/sean9999/transactor/transactor"
)

// CreateUserOp implements transactor.Op
var _ transactor.Op = (*CreateUserOp)(nil)

// CreateUserOp is the operation that represents creating a User.
type CreateUserOp struct {
	User     *User
	children []*address.CreateAddrOp // child transactors
	tx       *sql.Tx
	db       *sql.DB
}

func (op *CreateUserOp) Prepare(ctx context.Context) error {

	if op.tx != nil {
		return errors.New("there is already a transaction")
	}

	return transactor.RunOrCancel(ctx, func() error {
		return prepare(op)
	})
}

func (op *CreateUserOp) PrepareChildren(ctx context.Context) error {

	//	let's pretend we need to run these in sequence, for simplicity
	for _, child := range op.children {
		if err := child.Prepare(ctx); err != nil {
			return err
		}
	}

	addrs := make([]*address.Address, len(op.children))
	for i, child := range op.children {
		child.
	}

	return nil

}

func (op *CreateUserOp) Commit(ctx context.Context) error {

	if op.tx == nil {
		return errors.New("could not commit. nil transaction")
	}

	// application logic is simple for this case.
	// just turn around and tell the database "commit the transaction".
	fn := op.tx.Commit

	return transactor.RunOrCancel(ctx, fn)
}

func (op *CreateUserOp) Rollback() error {

	if op.tx == nil {
		return errors.New("there is no transaction to roll back")
	}

	err := op.tx.Rollback()
	if err != nil {
		return err
	}
	op.tx = nil

	return nil

}

func (op *CreateUserOp) Children() []transactor.Op {
	kids := make([]transactor.Op, len(op.children))
	for i, child := range op.children {
		kids[i] = child
	}
	return kids
}

func prepare(op *CreateUserOp) error {

	//	begin transaction
	tx, err := op.db.Begin()
	if err != nil {
		return err
	}
	op.tx = tx

	//	Add statements to our transaction.
	//	We need userId for subsequent dependant operations
	res, err := tx.Query(`
		INSERT INTO User (name, email) 
		VALUES ($1, $2) 
		RETURNING id`, op.User.Name, op.User.Email)
	if err != nil {
		return err
	}
	var userId int
	err = res.Scan(&userId)
	if err != nil {
		return err
	}
	if userId == 0 {
		return errors.New("0 User ID is wrong")
	}

	//	now we have our userId.
	op.User.ID = userId

	return nil

}

func BuildCreateUserOp(u *User, db *sql.DB) (*CreateUserOp, error) {

	//	things we know we need before we can even begin
	if u.Email == "" {
		return nil, errors.New("email is required")
	}

	op := new(CreateUserOp)
	op.User = u
	op.db = db
	op.children = make([]*address.CreateAddrOp, 0, len(u.Addresses))
	return op, nil
}
