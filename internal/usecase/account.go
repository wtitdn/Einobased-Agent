package usecase

import (
	"context"
	"einoproject/internal/repo"
)

type AccountService struct {
	accountRepo repo.AccountRepo
}

func NewAccountService(accountRepo repo.AccountRepo) *AccountService {
	return &AccountService{accountRepo: accountRepo}
}

func (a *AccountService) Login(ctx context.Context) error {
	return nil
}
