package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/sean9999/transactor/services/user"
	"github.com/sean9999/transactor"
)

func main() {

	//	this is our input. The operation should flesh it out.
	//	For example, User will get an ID from the database.
	//	Each address will get lat/long from a child operation.
	//usr := &user.User{
	//	Email: "frank@example.com",
	//	Addresses: []address.Address{
	//		address.Address{Street: "54 Tulane Hwy, Phoenix, Arizona"},
	//		address.Address{Street: "16 Parkway Ave, New York City, New York"},
	//	},
	//}

	db, err := sql.Open("ramsql", "users")
	if err != nil {
		log.Fatal(err)
	}
	op := user.NewCreateUserOp(db)

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

func prettyPrint(op *user.CreateUserOp) {
	data, err := json.MarshalIndent(op, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
