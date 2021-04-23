package main

import (
    "database/sql"
    "fmt"
    _ "github.com/mattn/go-sqlite3"
    "log"
)

const (
    DBFile         = "ip.db"
    CreateTableSql = `create table IF NOT EXISTS CFTestDetails (
    TestTime                datetime, 
    ASN                     int, 
    CITY                    text, 
    IP                      text, 
    LABEL                   text,
    ConnCount               int,
    ConnSuccessCount        int,
    ConnSuccessRate         float,
    ConnDurationAvg         float,
    ConnDurationMin         float,
    ConnDurationMax         float,
    DownloadCount           int,
    DownloadSuccessCount    int,
    DownloadSuccessRatio    float,
    DownloadSpeedAvg        float,
    DownloadSize            int,
    DownloadDurationSec     float)`
    InsertDataSqlExp = `insert into CFTestDetails (
    TestTime            , 
    ASN                 ,
    CITY                ,
    IP                  ,
    LABEL               ,
    ConnCount           ,
    ConnSuccessCount    ,
    ConnSuccessRate     ,
    ConnDurationAvg     ,
    ConnDurationMin     ,
    ConnDurationMax     ,
    DownloadCount       ,
    DownloadSuccessCount,
    DownloadSuccessRatio,
    DownloadSpeedAvg    ,
    DownloadSize        ,
    DownloadDurationSec  )
    values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
)

type CFTestDetail struct {
    TestTimeStr string
    ASN         int
    City        string
    Label       string
    VerifyResults
}

func openDB(dbFile string) *sql.DB {
    if len(dbFile) == 0 {
        dbFile = DBFile
    }
    db, err := sql.Open("sqlite3", dbFile)
    if err != nil {
        log.Fatal(fmt.Sprintf("%v\n", err))
    }
    return db
}

func dbExec(db *sql.DB, sql string, closeDB bool) *sql.Result {
    if closeDB {
        defer func() { _ = db.Close() }()
    }
    r, err := db.Exec(sql)
    if err != nil {
        log.Fatal(fmt.Sprintf("%v\n", err))
    }
    return &r
}


func openTable(dbFile string) *sql.DB {
    db := openDB(dbFile)
    _ = dbExec(db, CreateTableSql, false)
    return db
}

func QueryData(sql string, dbFile string) *[]CFTestDetail {
    db := openTable(dbFile)
    cfDetails := make([]CFTestDetail, 0)
    rows, err := db.Query(sql)
    if err != nil {
        log.Fatal(fmt.Sprintf("%v\n", err))
    }
    defer func() { _ = rows.Close() }()
    for rows.Next() {
        var tmpDetail CFTestDetail
        err = rows.Scan(&tmpDetail.TestTimeStr,
            &tmpDetail.ASN,
            &tmpDetail.City,
            &tmpDetail.IP,
            &tmpDetail.Label,
            &tmpDetail.PingCount,
            &tmpDetail.PingSuccessCount,
            &tmpDetail.PingSuccessRate,
            &tmpDetail.PingDurationAvg,
            &tmpDetail.PingDurationMin,
            &tmpDetail.PingDurationMax,
            &tmpDetail.DownloadCount,
            &tmpDetail.DownloadSuccessCount,
            &tmpDetail.DownloadSuccessRatio,
            &tmpDetail.DownloadSpeedAvg,
            &tmpDetail.DownloadSize,
            &tmpDetail.DownloadDurationSec)
        if err != nil {
            log.Fatal(err)
        }
        cfDetails = append(cfDetails, tmpDetail)
    }
    return &cfDetails
}

func InsertData(details []CFTestDetail, dbFile string) bool {
    if len(details) == 0 {
        return true
    }
    db := openTable(dbFile)
    tx, err := db.Begin()
    if err != nil {
        log.Fatal(err)
    }
    stmt, err := tx.Prepare(InsertDataSqlExp)
    if err != nil {
        log.Fatal(err)
    }
    defer func() {
        _ = stmt.Close()
    } ()
    for _, row := range details {
        _, err = stmt.Exec(
            &row.TestTimeStr,
            &row.ASN,
            &row.City,
            &row.IP,
            &row.Label,
            &row.PingCount,
            &row.PingSuccessCount,
            &row.PingSuccessRate,
            &row.PingDurationAvg,
            &row.PingDurationMin,
            &row.PingDurationMax,
            &row.DownloadCount,
            &row.DownloadSuccessCount,
            &row.DownloadSuccessRatio,
            &row.DownloadSpeedAvg,
            &row.DownloadSize,
            &row.DownloadDurationSec,
        )
        if err != nil {
            log.Fatal(err)
        }
    }
    err = tx.Commit()
    if err != nil {
        log.Fatal(err)
    }
    return true
}