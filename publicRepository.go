package publicRepository

import (
	"math"

	"gorm.io/gorm"
)

var DB *gorm.DB

type Pagination struct {
	Page  int
	Limit int
}

type QueryBuilder func(*gorm.DB) *gorm.DB

type IMainRepository[T any] interface {
	FindById(id *uint, queryFuncs ...QueryBuilder) (*T, error)
	FindAll(queryFuncs ...QueryBuilder) (*[]T, error)
	Create(m *T, queryFuncs ...QueryBuilder) (*T, error)
	Update(m *T, queryFuncs ...QueryBuilder) (*T, error)
	Delete(m *T, queryFuncs ...QueryBuilder) error
	FindAllPaginated(pagination *Pagination, queryFuncs ...QueryBuilder) (*[]T, int64, error)
	Count(queryFuncs ...QueryBuilder) (*int64, error)
	Exist(queryFuncs ...QueryBuilder) (*bool, error)
}

type MainRepository[T any] struct {
	db *gorm.DB
}

func NewMainRepository[T any]() IMainRepository[T] {
	return &MainRepository[T]{db: DB}
}

func (repo *MainRepository[T]) FindById(id *uint, queryFuncs ...QueryBuilder) (*T, error) {
	var result T
	q := repo.applyQueryBuilders(repo.db, queryFuncs)

	err := q.First(&result, id).Error
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (repo *MainRepository[T]) FindAll(queryFuncs ...QueryBuilder) (*[]T, error) {
	var results []T
	q := repo.applyQueryBuilders(repo.db, queryFuncs)

	err := q.Find(&results).Error
	if err != nil {
		return nil, err
	}

	return &results, nil
}

func (repo *MainRepository[T]) FindAllPaginated(pagination *Pagination, queryFuncs ...QueryBuilder) (*[]T, int64, error) {
	var results []T
	var count int64
	q := repo.applyQueryBuilders(repo.db, queryFuncs)

	offset := (pagination.Page - 1) * pagination.Limit

	err := q.Count(&count).Limit(pagination.Limit).Offset(offset).Find(&results).Error
	if err != nil {
		return nil, 0, err
	}

	maxPage := int64(math.Ceil(float64(count) / float64(pagination.Limit)))
	return &results, maxPage, nil
}

func (repo *MainRepository[T]) Create(m *T, queryFuncs ...QueryBuilder) (*T, error) {
	q := repo.applyQueryBuilders(repo.db, queryFuncs)

	err := q.Create(m).Error
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (repo *MainRepository[T]) Update(m *T, queryFuncs ...QueryBuilder) (*T, error) {
	q := repo.applyQueryBuilders(repo.db, queryFuncs)

	err := q.Save(m).Error
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (repo *MainRepository[T]) Delete(m *T, queryFuncs ...QueryBuilder) error {
	q := repo.applyQueryBuilders(repo.db, queryFuncs)

	return q.Delete(m).Error
}

func (repo *MainRepository[T]) Count(queryFuncs ...QueryBuilder) (*int64, error) {
	var count int64
	q := repo.applyQueryBuilders(repo.db, queryFuncs)

	err := q.Count(&count).Error
	if err != nil {
		return nil, err
	}

	return &count, nil
}

func (repo *MainRepository[T]) Exist(queryFuncs ...QueryBuilder) (*bool, error) {
	q := repo.applyQueryBuilders(repo.db, queryFuncs)

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, err
	}

	exists := count > 0
	return &exists, nil
}

func (repo *MainRepository[T]) applyQueryBuilders(q *gorm.DB, queryFuncs []QueryBuilder) *gorm.DB {
	for _, f := range queryFuncs {
		q = f(q)
	}
	return q
}