package repo

type Repository[ENTITY any, ID comparable] interface {
	Get(ID) (ENTITY, error)
	Update(entity ENTITY) error
	Insert(entity ENTITY) error
	Delete(id ID) error
}
