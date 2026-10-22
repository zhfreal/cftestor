package db

import (
	"time"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

const (
	DBFile    = "ip.db"
	TableName = "CFTD"
)

type DBRecord struct {
	TestTimeStr string  `gorm:"column:TestTime"`
	Asn         int     `gorm:"column:ASN"`
	City        string  `gorm:"column:CITY"`
	Loc         string  `gorm:"column:LOC"`
	IP          string  `gorm:"column:IP"`
	Label       string  `gorm:"column:LABEL"`
	DS          string  `gorm:"column:DS"`
	DTC         int     `gorm:"column:DTC"`
	DTPC        int     `gorm:"column:DTPC"`
	DTPR        float64 `gorm:"column:DTPR"`
	DA          float64 `gorm:"column:DA"`
	DMI         float64 `gorm:"column:DMI"`
	DMX         float64 `gorm:"column:DMX"`
	DLTC        int     `gorm:"column:DLTC"`
	DLTPC       int     `gorm:"column:DLTPC"`
	DLTPR       float64 `gorm:"column:DLTPR"`
	DLS         float64 `gorm:"column:DLS"`
	DLDS        int64   `gorm:"column:DLDS"`
	DLTD        float64 `gorm:"column:DLTD"`
}

func (a *DBRecord) TableName() string {
	return TableName
}

func OpenSqlite(dbFilePath string) (*gorm.DB, error) {
	dial := gormlite.Open(dbFilePath)
	gdb, err := gorm.Open(dial, &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now()
		},
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := gdb.DB()
	if err == nil {
		_, _ = sqlDB.Exec("PRAGMA journal_mode = WAL;")
		_, _ = sqlDB.Exec("PRAGMA synchronous = NORMAL;")
		_, _ = sqlDB.Exec("PRAGMA busy_timeout = 5000;")
	}
	return gdb, nil
}

func AddTableCFDT(db *gorm.DB) error {
	return db.AutoMigrate(&DBRecord{})
}

func AddCFDTRecords(db *gorm.DB, records []DBRecord) error {
	if len(records) == 0 {
		return nil
	}
	err := AddTableCFDT(db)
	if err != nil {
		return err
	}
	return db.Save(&records).Error
}
