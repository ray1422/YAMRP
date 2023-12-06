package client

// UserData consists of the token and id of the user.
type UserData struct {
	token string
	id    string
}

// Token returns the token of the user.
func (u *UserData) Token() string {
	return u.token
}

// ID returns the id of the user.
func (u *UserData) ID() string {
	return u.id
}
