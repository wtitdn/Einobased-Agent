package usecase

import (
	"context"
	"errors"
	"strings"

	"einoproject/internal/entity"
	"einoproject/internal/repo"
	pkgjwt "einoproject/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AccountService struct {
	accountRepo *repo.AccountRepo
}

func NewAccountService(accountRepo *repo.AccountRepo) *AccountService {
	return &AccountService{accountRepo: accountRepo}
}

type LoginResult struct {
	Account      *entity.Account
	Token        string
	RefreshToken string
}

var (
	ErrInvalidAccountInput = errors.New("username and password are required")
	ErrAccountExists       = errors.New("account already exists")
	ErrInvalidCredentials  = errors.New("invalid username or password")
)

func (a *AccountService) Register(ctx context.Context, username, password string) (*entity.Account, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidAccountInput
	}

	_, err := a.accountRepo.FindByUsername(ctx, username)
	if err == nil {
		return nil, ErrAccountExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	account := &entity.Account{
		Username: username,
		Password: string(hashedPassword),
	}
	if err := a.accountRepo.Register(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (a *AccountService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrInvalidAccountInput
	}

	account, err := a.accountRepo.FindByUsername(ctx, username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := pkgjwt.GenerateToken(account.ID, account.Username)
	if err != nil {
		return nil, err
	}
	refreshToken, err := pkgjwt.GenerateRefreshToken(account.ID)
	if err != nil {
		return nil, err
	}
	if err := a.accountRepo.Login(ctx, account.ID, token, refreshToken); err != nil {
		return nil, err
	}

	account.Token = token
	account.RefreshToken = refreshToken
	return &LoginResult{
		Account:      account,
		Token:        token,
		RefreshToken: refreshToken,
	}, nil
}
