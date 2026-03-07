package user

import (
	"database/sql"

	_ "github.com/proullon/ramsql/driver"
)

/**

User service is a simple service that exposes the database engine for use, and promises that a users table exists.

*/

var Database *sql.DB

func init() {

	Database.Exec(`
	CREATE TABLE users (
		user_id INT PRIMARY KEY AUTO_INCREMENT,
		name TEXT,
		email TEXT NOT NULL UNIQUE
	);
`)

}
