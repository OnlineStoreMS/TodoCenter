package seed

import (
	"log"

	"todocenter/internal/model"

	"gorm.io/gorm"
)

var defaultCategories = []struct {
	Name string
	Code string
	Sort int
}{
	{Name: "电商", Code: "ecommerce", Sort: 10},
	{Name: "发货", Code: "shipping", Sort: 20},
	{Name: "售后", Code: "aftersales", Sort: 30},
	{Name: "门店", Code: "store", Sort: 40},
}

func normalizeTenantID(id uint64) uint64 {
	if id == 0 {
		return 1
	}
	return id
}

// EnsureDefaultCategories inserts default categories for a tenant if missing.
func EnsureDefaultCategories(db *gorm.DB, tenantID uint64) error {
	tenantID = normalizeTenantID(tenantID)
	for _, def := range defaultCategories {
		var n int64
		if err := db.Model(&model.TodoCategory{}).
			Where("tenant_id = ? AND code = ?", tenantID, def.Code).
			Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		row := model.TodoCategory{
			TenantID: tenantID,
			Name:     def.Name,
			Code:     def.Code,
			Sort:     def.Sort,
			Enabled:  1,
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
		log.Printf("seed todo category tenant=%d name=%s code=%s", tenantID, def.Name, def.Code)
	}
	return nil
}
