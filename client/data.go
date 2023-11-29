package client

type UserData struct {
	token string
	id    string
}

func (u *UserData) Token() string {
	return u.token
}

func (u *UserData) ID() string {
	return u.id
}
