package services

import "context"

type DBRepositoryInterface interface {
	Ping(ctx context.Context) error
}

type DBService struct {
	dbRepo DBRepositoryInterface
}

func NewDBService(repository DBRepositoryInterface) *DBService {
	return &DBService{
		dbRepo: repository,
	}
}

func (db *DBService) CheckDBConnection(ctx context.Context) error {
	return db.dbRepo.Ping(ctx)
}
