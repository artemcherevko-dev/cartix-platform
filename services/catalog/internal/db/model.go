package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Attributes map[string]any

func (a Attributes) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Attributes) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
}

type Product struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Name        string     `json:"name" gorm:"size:255;not null"`
	Description string     `json:"description" gorm:"type:text"`
	PriceMinor  int64      `json:"price_minor" gorm:"not null"`
	Currency    string     `json:"currency" gorm:"size:10;not null"`
	Category    string     `json:"category" gorm:"size:50;not null;default:'general'"`
	Attributes  Attributes `json:"attributes" gorm:"type:jsonb"`
	Active      bool       `json:"active" gorm:"not null;default:true"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateInput struct {
	Name        string
	Description string
	PriceMinor  int64
	Currency    string
	Category    string
	Attributes  Attributes
	Active      bool
}
