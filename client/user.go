package client

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"time"
)

// UserModel provides the interface for user model containing user authentication and authorization.
type UserModel interface {
	// Token returns the token of the user for authentication.
	Token() string
	// UUID returns the UUID of the user for authorization.
	ID() string
}

// User is the implementation of UserModel.
type User struct {
	token string
	id    string
}

// NewUser allocates a user from server
func NewUser() *User {
	// TODO tells the server to create a new user
	timeStr := fmt.Sprintf("%d%d", time.Now().UnixMilli(), rand.Int())
	id := fmt.Sprintf("%x", md5.Sum([]byte(timeStr)))
	token := "token"
	return &User{
		token: token,
		id:    id,
	}
}

// Token returns the token of the user for authentication.
func (u *User) Token() string {
	return u.token
}

// ID returns the UUID of the user for authorization.
func (u *User) ID() string {
	return u.id
}
