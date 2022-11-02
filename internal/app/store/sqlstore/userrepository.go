package sqlstore

import (
	"database/sql"

	"github.com/SButnyakov/music-backend-authorization/internal/app/model"
	"github.com/SButnyakov/music-backend-authorization/internal/app/store"
)

type UserRepository struct {
	store *Store
}

func (r *UserRepository) Create(u *model.User) error {
	if err := u.Validate(); err != nil {
		return err
	}

	if err := u.BeforeCreate(); err != nil {
		return err
	}

	return r.store.db.QueryRow(
		"INSERT INTO users (login, encrypted_password, stage_name, cookie) OUTPUT Inserted.id VALUES (@p1, @p2, @p3, @p4)",
		u.Login,
		u.EncryptedPassword,
		u.StageName,
		u.AuthCookie,
		).Scan(&u.ID)
}

func (r *UserRepository) FindByLogin(login string) (*model.User, error) {
	u := &model.User{}
	if err := r.store.db.QueryRow(
		"SELECT id, login, encrypted_password, stage_name, cookie FROM users WHERE login = @p1",
		login,
	).Scan(
		&u.ID,
		&u.Login,
		&u.EncryptedPassword,
		&u.StageName,
		&u.AuthCookie,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}

		return nil, err
	}

	return u, nil
}

func (r *UserRepository) Find(id int) (*model.User, error) {
	u := &model.User{}
	if err := r.store.db.QueryRow(
		"SELECT id, login, encrypted_password, stage_name, cookie FROM users WHERE id = @p1",
		id,
	).Scan(
		&u.ID,
		&u.Login,
		&u.EncryptedPassword,
		&u.StageName,
		&u.AuthCookie,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrRecordNotFound
		}

		return nil, err
	}

	return u, nil
}

func (r *UserRepository) UpdateCookie(login string, cookie string) error {
	if err := r.store.db.QueryRow(
		"UPDATE users SET cookie = @p2 WHERE login = @p1",
		login,
		cookie,
		); err != nil {
		return err.Err()
	}
	return nil
}
