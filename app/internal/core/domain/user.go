package domain

import "time"

type User struct {
	Username  string    `db:"username"`
	Password  string    `db:"password"` // bcrypt hashed
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewUser(username, hashedPassword string) *User {
	now := time.Now()
	return &User{
		Username:  username,
		Password:  hashedPassword,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
