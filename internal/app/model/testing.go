package model

import "testing"

func TestUser(t *testing.T) *User {
	return &User{
		Login:    "login123",
		Password: "password",
	}
}
