package account

import (
	"errors"
	"testing"
)

func TestMenuServiceEarlyValidation(t *testing.T) {
	service := &MenuService{}

	if err := service.UpdateMenu(testUserCtx(), &MenuParam{}); !errors.Is(err, ErrMenuIDInvalid) {
		t.Fatalf("UpdateMenu expected ErrMenuIDInvalid, got %v", err)
	}
	if err := service.DeleteMenuById(testUserCtx(), 0); !errors.Is(err, ErrMenuIDInvalid) {
		t.Fatalf("DeleteMenuById expected ErrMenuIDInvalid, got %v", err)
	}
}
