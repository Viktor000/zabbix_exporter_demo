package main

import (
        "net/http"
        "fmt"
        "database/sql"
        _ "github.com/lib/pq"
        
        "github.com/prometheus/client_golang/prometheus"
        "github.com/prometheus/client_golang/prometheus/promauto"
        "github.com/prometheus/client_golang/prometheus/promhttp"
)
var db *sql.DB
const (
        host     = "________"
        port     = 5432
        user     = "postgres"
        password = "________"
        dbname   = "zab_25"
    )




var (    
        zabbix_alarm = promauto.NewGaugeVec(prometheus.GaugeOpts{
                Subsystem: "zabbix",
                Name: "alarm",
                Help: "The alarms from zabbix",
        },
        []string{"HostName","GroupName","Priority","EventId","AlarmMessage"},
        )
        )   


func CheckError(err error) {
        if err != nil {
            panic(err)
        }
    }

func retrieveRecord() {
        // connection string
        psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
        fmt.Println(psqlconn)
        // open database
        db, err := sql.Open("postgres", psqlconn)
        CheckError(err)
        
        // close database
        defer db.Close()
        var SQL_Q string
        //SQL_Q="SELECT eventid FROM problem"
        SQL_Q=`
        SELECT p.eventid AS eventid,
        p.name AS problemname,
    i.hostid AS ihostid,
    h.host AS hostname,
    hg.groupid,
    hgt.name AS groupname,
    t.priority AS tpriority,
    e.clock AS eventclock,
    e.acknowledged AS eacknowledged,
    SUM(CASE t.priority
        WHEN 5 THEN 100
        WHEN 4 THEN 100
        WHEN 3 THEN 1
        WHEN 2 THEN 1
        ELSE 0
    END) OVER (PARTITION BY hg.groupid) AS Group_Weight
FROM (((((((items i
    LEFT JOIN functions f ON ((i.itemid = f.itemid)))
    LEFT JOIN triggers t ON ((f.triggerid = t.triggerid)))
    RIGHT JOIN events e ON ((t.triggerid = e.objectid)))
    JOIN problem p ON ((e.eventid = p.eventid)))
    RIGHT JOIN hosts_groups hg ON ((i.hostid = hg.hostid)))
    RIGHT JOIN hstgrp hgt ON ((hg.groupid = hgt.groupid)))
    RIGHT JOIN hosts h ON ((i.hostid = h.hostid)))
WHERE ((i.status = 0) AND (p.r_clock = 0))
        `
        rows, err := db.Query(SQL_Q)
        CheckError(err)
        defer rows.Close()
        for rows.Next() {
                    var eventid string
                    var problemname string
                    var hostid int
                    var hostname string
                    var groupid int
                    var groupname string
                    var priority string
                    var eventclock int
                    var eacknowledged float64
                    var Group_Weight int          
                err = rows.Scan(&eventid,&problemname,&hostid, &hostname,&groupid, &groupname, &priority, &eventclock,&eacknowledged,&Group_Weight)
                //err = rows.Scan(&ev)
                CheckError(err)
                
                //fmt.Println(eventid, hostid, hostname,groupid,groupname,priority,eventclock,eacknowledged,Group_Weight)
                zabbix_alarm.With(prometheus.Labels{"HostName":hostname,"GroupName":groupname,"Priority":priority,"EventId":eventid,"AlarmMessage":problemname}).Set(eacknowledged) 
                }
        // check db
        err = db.Ping()
        CheckError(err)

        fmt.Println("Connected!")  
 }

 
CheckError(err)

}

func main() {
        retrieveRecord()
        http.Handle("/metrics", promhttp.Handler())
        http.ListenAndServe(":2112", nil)
}
