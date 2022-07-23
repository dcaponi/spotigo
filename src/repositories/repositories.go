package repositories

import (
	"context"

	"github.com/dcaponi/spotigo/src/app_error"
	"github.com/dcaponi/spotigo/src/domain"
	"github.com/dcaponi/spotigo/src/repositories/db_entities"
	"gorm.io/gorm"
)

type repositories struct {
	DB *gorm.DB
}

type Repositories interface {
	CreateUser(ctx context.Context, domainUser domain.User) error
	SearchUsers(ctx context.Context) ([]domain.User, error)
}

func NewRepository(db *gorm.DB) Repositories {
	return repositories{
		DB: db,
	}
}

func (repo repositories) CreateUser(ctx context.Context, domainUser domain.User) error {
	user := db_entities.NewUserFromDomain(domainUser)
	result := repo.DB.FirstOrCreate(&user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return app_error.UserAlreadyExists
	}
	return nil
}

func (repo repositories) SearchUsers(ctx context.Context) ([]domain.User, error) {
	users := db_entities.Users{}
	if err := repo.DB.Find(&users).Error; err != nil {
		return []domain.User{}, err
	}
	return users.ToDomain(), nil
}
