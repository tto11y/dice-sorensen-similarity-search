package auth

import (
	"context"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/models"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

const usernamePasswordFalse = "username or password false"

type AuthService struct {
	*environment.Env
}

// DoLogin gets the login credentials for <user>
func (c *AuthService) DoLogin(user *models.User) error {
	var foundUser models.User

	err := c.FindUserLoginCredentials(context.Background(), user.Username, &foundUser)
	if err != nil {
		return errors.New(usernamePasswordFalse)
	}
	err = models.VerifyPassword(foundUser.Password, user.Password)
	if err != nil && errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return errors.New(usernamePasswordFalse)
	}
	user.ID = foundUser.ID
	return nil
}
