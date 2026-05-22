package system

import (
	"errors"
	"testing"
)

func TestDictServiceEarlyValidation(t *testing.T) {
	service := &DictService{}

	if _, err := service.GetDictList(testUserCtx(), &DictListCondition{StartTime: "bad-time"}); !errors.Is(err, ErrTimeFormat) {
		t.Fatalf("GetDictList expected ErrTimeFormat, got %v", err)
	}
	if _, err := service.UpdateDict(testUserCtx(), &DictParam{}); !errors.Is(err, ErrDictCodeNotFound) {
		t.Fatalf("UpdateDict expected ErrDictCodeNotFound, got %v", err)
	}
	if err := service.DeleteById(testUserCtx(), 0); !errors.Is(err, ErrDictCodeNotFound) {
		t.Fatalf("DeleteById expected ErrDictCodeNotFound, got %v", err)
	}
	if _, err := service.GetItemListByCode(testUserCtx(), ""); !errors.Is(err, ErrDictCodeNotFound) {
		t.Fatalf("GetItemListByCode expected ErrDictCodeNotFound, got %v", err)
	}
	if _, err := service.UpdateItem(testUserCtx(), &DictItemParam{}); !errors.Is(err, ErrDictCodeItemNotFound) {
		t.Fatalf("UpdateItem expected ErrDictCodeItemNotFound, got %v", err)
	}
	if err := service.DeleteItemById(testUserCtx(), 0); !errors.Is(err, ErrDictCodeItemNotFound) {
		t.Fatalf("DeleteItemById expected ErrDictCodeItemNotFound, got %v", err)
	}
}
