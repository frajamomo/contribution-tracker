package application

import (
	"context"
	"time"

	"contribution-tracker/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	accounts  UserAccountRepository
	users     UserRepository
	teams     TeamRepository
	jwtSecret []byte
}

func NewAuthService(accounts UserAccountRepository, users UserRepository, teams TeamRepository, jwtSecret []byte) *AuthService {
	return &AuthService{accounts: accounts, users: users, teams: teams, jwtSecret: jwtSecret}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*AuthToken, error) {
	account, err := s.accounts.FindByUsername(ctx, username)
	if err != nil {
		return nil, NewUnauthorizedError("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return nil, NewUnauthorizedError("invalid credentials")
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	roleStrings := make([]string, 0, len(account.Roles))
	for r := range account.Roles {
		roleStrings = append(roleStrings, string(r))
	}

	claims := jwt.MapClaims{
		"sub":   account.ID,
		"roles": roleStrings,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, NewInternalError("failed to sign token", err)
	}

	return &AuthToken{
		Value:     tokenStr,
		ExpiresAt: expiresAt,
		AccountID: account.ID,
		Roles:     account.Roles,
	}, nil
}

func (s *AuthService) Validate(tokenStr string) (*AuthContext, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, NewUnauthorizedError("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, NewUnauthorizedError("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, NewUnauthorizedError("invalid claims")
	}

	accountID, _ := claims.GetSubject()
	if accountID == "" {
		return nil, NewUnauthorizedError("missing subject")
	}

	roles := make(map[domain.Role]bool)
	if roleList, ok := claims["roles"].([]interface{}); ok {
		for _, r := range roleList {
			if rs, ok := r.(string); ok {
				roles[domain.Role(rs)] = true
			}
		}
	}

	account, err := s.accounts.FindByID(context.Background(), accountID)
	if err != nil {
		return nil, NewUnauthorizedError("account not found")
	}

	user, err := s.users.FindByAccountID(context.Background(), accountID)
	if err != nil {
		return nil, NewUnauthorizedError("user not found")
	}

	return &AuthContext{
		AccountID: account.ID,
		UserID:    user.ID,
		Roles:     roles,
	}, nil
}
