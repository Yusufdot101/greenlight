package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"github.com/Yusufdot101/greenlight/internal/validator"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

func generateToken(userID int64, timeToLive time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(timeToLive),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, plaintext string) {
	v.CheckAdd(plaintext != "", "token", "must be provided")
	v.CheckAdd(len(plaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModel struct {
	DB *sql.DB
}

func (model TokenModel) NewToken(
	userID int64, timeToLive time.Duration, scope string) (*Token, error,
) {
	token, err := generateToken(userID, timeToLive, scope)
	if err != nil {
		return nil, err
	}

	err = model.InsertToken(token)

	return token, err
}

func (model *TokenModel) InsertToken(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)
	`
	args := []any{
		token.Hash,
		token.UserID,
		token.Expiry,
		token.Scope,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := model.DB.ExecContext(ctx, query, args...)

	return err
}

func (model *TokenModel) DeleteAllForUser(userID int64, scope string) error {
	query := `
		DELETE FROM tokens
		WHERE user_id = $1 AND scope = $2
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := model.DB.ExecContext(ctx, query, userID, scope)

	return err
}
