package teststore_test

import (
	"testing"

	"github.com/SButnyakov/music-backend-authorization/internal/app/model"
	"github.com/SButnyakov/music-backend-authorization/internal/app/store"
	teststore "github.com/SButnyakov/music-backend-authorization/internal/app/store/teststore"
	"github.com/stretchr/testify/assert"
)

func TestUserRepository_Create(t *testing.T) {
	s := teststore.New()
	u := model.TestUser(t)
	assert.NoError(t, s.User().Create(u))
	assert.NotNil(t, u)
}

func TestUserRepository_FindByLogin(t *testing.T) {
	s := teststore.New()
	login := "login"
	_, err := s.User().FindByLogin(login)
	assert.EqualError(t, err, store.ErrRecordNotFound.Error())

	u := model.TestUser(t)
	u.Login = login
	s.User().Create(u)
	u, err = s.User().FindByLogin(login)
	assert.NoError(t, err)
	assert.NotNil(t, u)
}

func TestUserRepository_Find(t *testing.T) {
	s := teststore.New()
	u1 := model.TestUser(t)
	s.User().Create(u1)
	u2, err := s.User().Find(u1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, u2)
}
