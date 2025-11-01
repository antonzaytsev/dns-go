package postgres

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"
)

// DNSLog represents a DNS log entry in the database
type DNSLog struct {
	ID                  uint        `gorm:"primaryKey;autoIncrement"`
	UUID                string      `gorm:"type:varchar(255);uniqueIndex;not null"`
	Timestamp           time.Time   `gorm:"type:timestamp;not null;index"`
	ClientIP            string      `gorm:"type:inet;not null;index"`
	Query               string      `gorm:"type:varchar(255);not null;index"`
	QueryType           string      `gorm:"type:varchar(10);not null;index"`
	QueryID             *int        `gorm:"type:integer"`
	Status              string      `gorm:"type:varchar(50);not null;index"`
	CacheHit            bool        `gorm:"default:false;index"`
	DurationMs          *float64    `gorm:"type:double precision"`
	ResponseUpstream    *string     `gorm:"type:varchar(255);index"`
	ResponseRcode       *string     `gorm:"type:varchar(10)"`
	ResponseAnswerCount *int        `gorm:"type:integer"`
	ResponseRTTMs       *float64    `gorm:"type:double precision"`
	Upstreams           JSONB       `gorm:"type:jsonb"`
	Answers             JSONB       `gorm:"type:jsonb"`
	IPAddresses         StringArray `gorm:"type:inet[]"`
	CreatedAt           time.Time   `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for DNSLog
func (DNSLog) TableName() string {
	return "dns_logs"
}

// DNSMapping represents a DNS mapping entry in the database
type DNSMapping struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Domain    string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	IPAddress string    `gorm:"type:varchar(45);not null"`
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for DNSMapping
func (DNSMapping) TableName() string {
	return "dns_mappings"
}

// JSONB is a custom type for PostgreSQL JSONB fields
// It can represent both objects and arrays
type JSONB []interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil || len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		// Try to get bytes from string
		if str, ok := value.(string); ok {
			bytes = []byte(str)
		} else {
			return nil
		}
	}

	return json.Unmarshal(bytes, j)
}

// StringArray is a custom type for PostgreSQL string arrays
type StringArray []string

// Value implements driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	// Return as slice - GORM/pgx will handle conversion
	return []string(s), nil
}

// Scan implements sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	// PostgreSQL driver (pgx) returns arrays as []string directly
	if arr, ok := value.([]string); ok {
		*s = StringArray(arr)
		return nil
	}

	// Fallback: try to parse from []byte
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	// Parse PostgreSQL array format {val1,val2,val3}
	str := string(bytes)
	if len(str) < 2 {
		*s = []string{}
		return nil
	}

	// Remove curly braces
	str = str[1 : len(str)-1]
	if str == "" {
		*s = []string{}
		return nil
	}

	// Simple split
	parts := strings.Split(str, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, `"`)
		result = append(result, part)
	}

	*s = StringArray(result)
	return nil
}
