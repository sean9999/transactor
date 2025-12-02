package address

import (
	"context"
	"database/sql"

	"github.com/sean9999/transactor/repo"
)

var _ repo.Repository[*Address] = (*Repo)(nil)

type Repo struct {
	db *sql.DB
}

func NewAddressRepo() (*Repo, error) {

	db, err := sql.Open("ramsql", "addresses")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
		CREATE TABLE addresses (
		    id SERIAL PRIMARY KEY,
		    userId INTEGER NOT NULL,
		    street TEXT NOT NULL,
		    lat FLOAT NOT NULL,
		    long FLOAT NOT NULL
		);`)
	if err != nil {
		return nil, err
	}
	return &Repo{db: db}, nil
}

func (repo *Repo) GetByUser(_ context.Context, userId int) ([]Address, error) {
	rows, err := repo.db.Query(`SELECT * FROM addresses WHERE userId = ?`, userId)
	if err != nil {
		return nil, err
	}
	addresses := make([]Address, 0)
	err = rows.Scan(addresses)
	if err != nil {
		return nil, err
	}
	return addresses, nil
}

func (repo *Repo) Get(ctx context.Context, id string) (*Address, error) {
	u := new(Address)
	err := repo.db.QueryRowContext(ctx, `SELECT * FROM users WHERE id = $1`, id).Scan(u)

	return u, err
}

func (repo *Repo) Set(ctx context.Context, u *User) error {

	statement := `
		INSERT INTO users (Name, Email)
		VALUES ($1, $2)
		WHERE id = $3;`

	_, err := repo.db.ExecContext(ctx, statement, u.Name, u.Email, u.ID)
	return err
}

func (repo *Repo) Delete(ctx context.Context, id string) error {

	statement := `
		DELETE FROM users
		WHERE id = $1;`

	_, err := repo.db.ExecContext(ctx, statement, id)
	return err
}
