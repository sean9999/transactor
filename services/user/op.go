package user

import (
	"context"
	"database/sql"
	"errors"
	"iter"

	"github.com/sean9999/transactor/services/address"
	"github.com/sean9999/transactor/transactor"
)

// CreateUserOp implements transactor.Op
var _ transactor.Op = (*CreateUserOp)(nil)

func NewCreateUserOp(db *sql.DB) *CreateUserOp {
	uop := new(CreateUserOp)
	uop.db = db
	uop.children = make([]*address.CreateAddrsOp, 0)
	return uop
}

func (op *CreateUserOp) Initialize(args ...any) error {
	if len(args) != 2 {
		return errors.New("expected 2 arguments: name and email")
	}
	op.Name = args[0].(string)
	op.Email = args[1].(string)
	return nil
}

// CreateUserOp is the operation that represents creating a User.
type CreateUserOp struct {
	UserId   int64
	Name     string
	Email    string
	children []*address.CreateAddrsOp // child transactors
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

func (op *CreateUserOp) Children() iter.Seq[transactor.Op] {
	return transactor.AsOps(op.children)
}

func prepare(op *CreateUserOp) error {

	//	begin transaction
	tx, err := op.db.Begin()
	if err != nil {
		return err
	}
	op.tx = tx

	//	Add statements to our transaction.
	//	We need userId for later dependent operations
	res, err := tx.Query(`
		INSERT INTO User (name, email) 
		VALUES ($1, $2) 
		RETURNING id`, op.Name, op.Email)
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
	op.UserId = int64(userId)

	return nil

}
