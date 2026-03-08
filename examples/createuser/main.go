package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/proullon/ramsql/driver"
	"github.com/sean9999/transactor"
)

func main() {

	db, err := sql.Open("ramsql", "users")
	if err != nil {
		log.Fatal(err)
	}
	op := NewCreateUserOp(db)

	op.Initialize("John Doe", "yes@yes.com")

	op.Name = "John Doe"
	op.Email = "yes@yes.com"
	trans := transactor.NewTransactor(op)

	ctx := context.Background()

	//	do the work
	err = trans.Transact(ctx)
	if err != nil {
		log.Fatal(err)
	}

	prettyPrint(op)

}

func prettyPrint(op *CreateUserOp) {
	data, err := json.MarshalIndent(op, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
