package xdatabase_test

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"

	"snowgo/pkg/xdatabase"
)

func TestNewBaseRepository(t *testing.T) {
	db := &gorm.DB{}
	dbMap := map[string]*gorm.DB{"extra": {}}
	repo := xdatabase.NewBaseRepository(db, dbMap)
	if repo == nil {
		t.Fatal("NewBaseRepository returned nil")
	}
}

func TestGetDB(t *testing.T) {
	extraDB := &gorm.DB{}
	dbMap := map[string]*gorm.DB{"extra": extraDB}

	repo := xdatabase.NewBaseRepository(&gorm.DB{}, dbMap)

	t.Run("happy: existing db", func(t *testing.T) {
		got, err := repo.GetDB("extra")
		if err != nil {
			t.Fatalf("GetDB error: %v", err)
		}
		if got != extraDB {
			t.Fatal("GetDB returned wrong db")
		}
	})

	t.Run("error: nil dbMap", func(t *testing.T) {
		r := xdatabase.NewBaseRepository(&gorm.DB{}, nil)
		_, err := r.GetDB("any")
		if err == nil {
			t.Fatal("expected error for nil dbMap")
		}
	})

	t.Run("error: missing key", func(t *testing.T) {
		_, err := repo.GetDB("nonexistent")
		if err == nil {
			t.Fatal("expected error for missing key")
		}
	})

	t.Run("error: nil db value in map", func(t *testing.T) {
		r := xdatabase.NewBaseRepository(&gorm.DB{}, map[string]*gorm.DB{"empty": nil})
		_, err := r.GetDB("empty")
		if err == nil {
			t.Fatal("expected error for nil db value")
		}
	})
}

func TestGetBaseRepository(t *testing.T) {
	extraDB := &gorm.DB{}
	dbMap := map[string]*gorm.DB{"extra": extraDB}
	repo := xdatabase.NewBaseRepository(&gorm.DB{}, dbMap)

	t.Run("happy: existing db", func(t *testing.T) {
		sub, err := repo.GetBaseRepository("extra")
		if err != nil {
			t.Fatalf("GetBaseRepository error: %v", err)
		}
		if sub == nil {
			t.Fatal("GetBaseRepository returned nil")
		}
		// Verify sub-repo can find the same db
		got, err := sub.GetDB("extra")
		if err != nil {
			t.Fatalf("sub.GetDB error: %v", err)
		}
		if got != extraDB {
			t.Fatal("GetBaseRepository has wrong db")
		}
	})

	t.Run("error: missing db", func(t *testing.T) {
		_, err := repo.GetBaseRepository("nonexistent")
		if err == nil {
			t.Fatal("expected error for missing db")
		}
	})
}

func TestUse(t *testing.T) {
	db := &gorm.DB{}
	repo := xdatabase.NewBaseRepository(db, nil)
	readRepo := repo.Use(dbresolver.Read)
	if readRepo == nil {
		t.Fatal("Use returned nil")
	}
}
