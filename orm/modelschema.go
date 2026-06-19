package orm

import (
	ninja "github.com/shijl0925/gin-ninja"
	"gorm.io/gorm"
)

// ApplyModelSchemaPreloads applies the GORM preload paths implied by a model
// schema descriptor. It is useful when a response schema uses Depth and the
// query should load the same relation graph that will be serialized.
func ApplyModelSchemaPreloads[T any](db *gorm.DB, descriptor ninja.ModelSchemaDescriptor[T]) *gorm.DB {
	return applyPreloads(db, descriptor.Preloads())
}

// ApplyResponseModelPreloads applies the GORM preload paths implied by a
// response schema type that embeds ninja.ModelSchema.
func ApplyResponseModelPreloads[T any](db *gorm.DB) *gorm.DB {
	return applyPreloads(db, ninja.ModelSchemaPreloads[T]())
}

func applyPreloads(db *gorm.DB, preloads []string) *gorm.DB {
	if db == nil {
		return nil
	}
	for _, preload := range preloads {
		db = db.Preload(preload)
	}
	return db
}
