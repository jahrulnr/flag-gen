package di

import (
	"flag-gen/domain/repository"
	mysqlrepo "flag-gen/infrastructure/repository/mysql"
)

func ProvideAdminReadRepository(db *mysqlrepo.DB) repository.AdminReadRepository {
	return mysqlrepo.NewAdminReadRepository(db)
}
