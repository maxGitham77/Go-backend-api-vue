package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"time"
)

const dbTimeout = time.Second * 3

var db *sql.DB

func New(dbPool *sql.DB) Models {
	db = dbPool

	return Models{
		User:   User{},
		Token:  Token{},
		Book:   Book{},
		Author: Author{},
	}
}

type Models struct {
	User   User
	Token  Token
	Book   Book
	Author Author
}

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Firstname string    `json:"first_name,omitempty"`
	Lastname  string    `json:"last_name,omitempty"`
	Password  string    `json:"password"`
	Active    int       `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Token     Token     `json:"token"`
}

func (u *User) GetAll() ([]*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, email, first_name, last_name, password, user_active, created_at, updated_at, 
	case
		when(select count(id) from tokens t where user_id = users.id and t.expiry > NOW()) > 0 then 1
		else 0
	end as has_token
	from users order by last_name`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User

	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Firstname,
			&user.Lastname,
			&user.Password,
			&user.Active,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.Token.ID,
		)
		if err != nil {
			return nil, err
		}

		users = append(users, &user)
	}
	return users, nil
}

func (u *User) GetByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, email, first_name, last_name, password, user_active, created_at, updated_at from users where email = $1`

	var user User
	row := db.QueryRowContext(ctx, query, email)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Firstname,
		&user.Lastname,
		&user.Password,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *User) GetById(id int) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, email, first_name, last_name, password, user_active, created_at, updated_at from users where id = $1`

	var user User
	row := db.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Firstname,
		&user.Lastname,
		&user.Password,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *User) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `update users set
		 email = $1,
		 first_name = $2,
		 last_name = $3,
		 user_active = $4,
		 updated_at = $5
		 where id = $6`

	_, err := db.ExecContext(ctx, stmt,
		u.Email,
		u.Firstname,
		u.Lastname,
		u.Active,
		time.Now(),
		u.ID,
	)

	if err != nil {
		return err
	}

	return nil
}

// Delete delets one user from the database, by ID
func (u *User) Delete() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `delete from users where id = $1`

	_, err := db.ExecContext(ctx, stmt, u.ID)
	if err != nil {
		return err
	}

	return nil
}

func (u *User) DeleteByID(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `delete from users where id = $1`

	_, err := db.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	return nil
}

// Inser inserts a new user into the database, and returns the ID of the newly inserted row
func (u *User) Insert(user User) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
	if err != nil {
		return 0, err
	}

	var newID int
	stmt := `insert into users(email, first_name, last_name, password, user_active, created_at, updated_at)
	values ($1, $2, $3, $4, $5, $6, $7) returning id
`
	err = db.QueryRowContext(ctx, stmt,
		user.Email,
		user.Firstname,
		user.Lastname,
		hashedPassword,
		user.Active,
		time.Now(),
		time.Now(),
	).Scan(&newID)

	if err != nil {
		return 0, err
	}
	return newID, nil
}

func (u *User) ResetPassword(password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	stmt := `update users set password = $1 where id = $2`
	_, err = db.ExecContext(ctx, stmt, hashedPassword, u.ID)

	if err != nil {
		return err
	}

	return nil

}

func (u *User) PasswordMatches(plainText string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainText))

	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			// invalid password
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

type Token struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	TokenHash []byte    `json:"_"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated-at"`
	Expiry    time.Time `json:"expiry"`
}

func (t *Token) GetByToken(plainText string) (*Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, user_id, email, token, token_hash, created_at, updated_at, expiry from tokens where token = $1`

	var token Token
	row := db.QueryRowContext(ctx, query, plainText)
	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.Email,
		&token.Token,
		&token.TokenHash,
		&token.CreatedAt,
		&token.UpdatedAt,
		&token.Expiry,
	)

	if err != nil {
		return nil, err
	}

	return &token, nil

}

func (t *Token) GetUserForToken(token Token) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, email, first_name, last_name, password, user_active, created_at, updated_at from users where id = $1`

	var user User
	row := db.QueryRowContext(ctx, query, token.UserID)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Firstname,
		&user.Lastname,
		&user.Password,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil

}

func (t *Token) GenerateToken(userID int, ttl time.Duration) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Token = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Token))
	token.TokenHash = hash[:]

	return token, nil
}

// AuthenticateToken takes the full http request, extract the authorization header,
// takes the plain text token from that header and looks up the associated token entry in the database,
// and then finds the user associated with that token. If the token is valid and a user is found,
// the user is returned; otherwise, it returns an error
func (t *Token) AuthenticateToken(r *http.Request) (*User, error) {
	// get the authorization header
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("no authorization header received")
	}

	// get the plain text token from the header
	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("no authorization header received and headerParts")
	}

	token := headerParts[1]

	// make sure the token is of the correct length
	if len(token) != 26 {
		return nil, errors.New("Wrong token size")
	}

	// get the token from the database, using the plain text token to find it
	tokenFromDb, err := t.GetByToken(token)
	if err != nil {
		return nil, errors.New("no matching token found")
	}

	// Make sure the token has not expired
	if tokenFromDb.Expiry.Before(time.Now()) {
		return nil, errors.New("Expired token")
	}

	// Gt the user associated with the token
	user, err := t.GetUserForToken(*tokenFromDb)
	if err != nil {
		return nil, errors.New("No matching user found")
	}

	if user.Active == 0 {
		return nil, errors.New("user not active")
	}

	return user, nil
}

func (t *Token) Insert(token Token, u User) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// delete any existing tokens
	stmt := `delete from tokens where user_id = $1`
	_, err := db.ExecContext(ctx, stmt, token.UserID)
	if err != nil {
		return err
	}

	token.Email = u.Email

	stmt = `insert into tokens (user_id, email, token, token_hash, created_at, updated_at, expiry) 
	values ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.ExecContext(ctx, stmt,
		token.UserID,
		token.Email,
		token.Token,
		token.TokenHash,
		time.Now(),
		time.Now(),
		token.Expiry,
	)
	if err != nil {
		return err
	}
	return nil
}

// DeleteByToken deletes a token, by plain text token
func (t *Token) DeleteByToken(plainText string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `delete from tokens where token = $1`

	_, err := db.ExecContext(ctx, stmt, plainText)
	if err != nil {
		return err
	}

	return nil
}

func (t *Token) DeleteTokensForUser(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := "delete from tokens where user_id = $1"
	_, err := db.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}
	return nil
}

// ValidToken makes certain that a given token is valid. In order to be valid the token
// must exist in the database, the associated user must exist in the database and the token
// must not have expired
func (t *Token) ValidToken(plainText string) (bool, error) {
	token, err := t.GetByToken(plainText)
	if err != nil {
		return false, errors.New("no matching token found")
	}

	_, err = t.GetUserForToken(*token)
	if err != nil {
		return false, errors.New("no matching user found")
	}

	if token.Expiry.Before(time.Now()) {
		return false, errors.New("expired token")
	}

	return true, nil
}
