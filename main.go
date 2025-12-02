package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/sean9999/transactor/services/address"
	"github.com/sean9999/transactor/services/user"
	"github.com/sean9999/transactor/transactor"
)

func main() {

	//	this is our input. The operation should flesh it out.
	//	For example, User will get an ID from the database.
	//	Each address will get lat/long from a child operation.
	usr := &user.User{
		Email: "frank@example.com",
		Addresses: []address.Address{
			address.Address{Street: "54 Tulane Hwy, Phoenix, Arizona"},
			address.Address{Street: "16 Parkway Ave, New York City, New York"},
		},
	}

	db, err := sql.Open("ramsql", "users")
	if err != nil {
		log.Fatal(err)
	}

	op, err := user.BuildCreateUserOp(usr, db)
	if err != nil {
		log.Fatal(err)
	}

	tran := transactor.NewTransactor(op)

	ctx := context.Background()

	//	do the work
	err = tran.Transact(ctx)
	if err != nil {
		log.Fatal(err)
	}

	prettyPrint(usr)

}

func prettyPrint(u *user.User) {
	data, err := json.MarshalIndent(u, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
