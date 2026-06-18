package repo

import (
	"context"
	"einoproject/internal/entity"

	"gorm.io/gorm"
)

type AccountRepo struct {
	db *gorm.DB
}

func NewAccountRepo(db *gorm.DB) *AccountRepo {
	return &AccountRepo{db: db}
}

func (a *AccountRepo) FindByUsername(ctx context.Context, username string) (*entity.Account, error) {
	var account entity.Account
	if err := a.db.WithContext(ctx).Where("username = ?", username).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (a *AccountRepo) Login(ctx context.Context, id uint, token, refreshToken string) error {
	if err := a.db.WithContext(ctx).Model(&entity.Account{}).Where("id = ?", id).Updates(map[string]interface{}{"token": token, "refresh_token": refreshToken}).Error; err != nil {
		return err
	}
	return nil
}
func (a *AccountRepo) Register(ctx context.Context, account *entity.Account) error {
	if err := a.db.WithContext(ctx).Create(account).Error; err != nil {
		return err
	}
	return nil
}
