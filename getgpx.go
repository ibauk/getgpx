package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/flopp/go-coordsparser"
	_ "github.com/mattn/go-sqlite3"
)

var SourceDBName = flag.String("db", "", "ScoreMaster format database containing GPX info")
var ExternalMapLink = flag.String("map", "https://www.google.co.uk/maps/search/", "Url of external mapping service")
var NoMapLink = flag.Bool("nolink", false, "Suppress external map link")
var Symbol2Use = flag.String("symbol", "Circle, Green", "Spec of symbol for waypoints")
var OutputGPX = flag.String("gpx", "waypoints.gpx", "Name of output GPX")

const apptitle = "getgpx v0.1"

const gpxheader = `<?xml version="1.0" encoding="utf-8"?>
<gpx creator="Bob Stammers (` + apptitle + `)" version="1.1"
xsi:schemaLocation="http://www.topografix.com/GPX/1/1 
http://www.topografix.com/GPX/1/1/gpx.xsd" 
xmlns="http://www.topografix.com/GPX/1/1" 
xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
`

var DBH *sql.DB
var GPXF *os.File
var RallyTitle string

func fileExists(x string) bool {

	_, err := os.Stat(x)
	//return !errors.Is(err, os.ErrNotExist)
	return err == nil

}

func xmlsafe(s string) string {

	x := map[string]string{`&`: `&amp;`, `"`: `&quot;`, `<`: `&lt;`, `>`: `&gt;`, `'`: `&#39;`}
	res := s
	for k, v := range x {
		res = strings.ReplaceAll(res, k, v)
	}
	return res
}

func writeWaypoint(lat, lon float64, bonusid, briefdesc string) {

	wpt := fmt.Sprintf("<wpt lat=\"%v\" lon=\"%v\"><name>%v", lat, lon, xmlsafe(bonusid))
	wpt += fmt.Sprintf("-%v", xmlsafe(briefdesc))
	wpt += "</name>"
	GPXF.WriteString(wpt)
	if RallyTitle != "" {
		GPXF.WriteString(fmt.Sprintf("<cmt>%v</cmt>", xmlsafe(RallyTitle)))
	}
	if *ExternalMapLink != "" && !*NoMapLink {
		GPXF.WriteString(fmt.Sprintf(`<link href="%v%v,%v" />`, *ExternalMapLink, lat, lon))
	}
	if *Symbol2Use != "" {
		GPXF.WriteString(fmt.Sprintf("<sym>%v</sym>", *Symbol2Use))
	}
	GPXF.WriteString("</wpt>\n")

}

func completeGPX() {

	GPXF.WriteString("</gpx>\n")

}

func main() {

	var err error
	flag.Parse()

	fmt.Printf("%v Copyright (c) 2023 Bob Stammers\nI extract bonus coordinates from a ScoreMaster database into a GPX file\n\n", apptitle)
	if *SourceDBName == "" {
		fmt.Println("You must specify the database to use, -db path")
		return
	}
	if !fileExists(*SourceDBName) {
		fmt.Printf("The database %v does not exist\n", *SourceDBName)
		return
	}
	DBH, err = sql.Open("sqlite3", *SourceDBName)
	if err != nil {
		panic(err)
	}
	defer DBH.Close()

	sqlx := "SELECT RallyTitle FROM rallyparams"
	rows, err := DBH.Query(sqlx)
	if err != nil {
		fmt.Printf("ERROR! %v\nproduced %v\n", sqlx, err)
		return
	}
	RallyTitle = ""
	if rows.Next() {
		rows.Scan(&RallyTitle)
	}
	rows.Close()

	if *OutputGPX == "" {
		fmt.Println("You must specify the GPX to create, -gpx path")
		return
	}

	fmt.Printf("Generating GPX %v\n", *OutputGPX)
	GPXF, _ = os.Create(*OutputGPX)
	defer GPXF.Close()
	GPXF.WriteString(gpxheader)

	generateWaypoints()

	completeGPX()

}

func generateWaypoints() {

	var bonus, title, coords string

	sqlx := "SELECT BonusID, BriefDesc, Coords FROM bonuses WHERE Coords IS NOT NULL AND Coords <> ''"
	sqlx += " ORDER BY BonusID"

	rows, err := DBH.Query(sqlx)
	if err != nil {
		fmt.Printf("ERROR! %v\nproduced %v\n", sqlx, err)
		return
	}
	defer rows.Close()
	var bonusCount, badCoordCount int
	for rows.Next() {
		rows.Scan(&bonus, &title, &coords)
		bonusCount++
		Lat, Lon, err := coordsparser.Parse(strings.ReplaceAll(strings.ReplaceAll(coords, "°", " "), "'", " "))
		if err != nil {
			badCoordCount++
			continue // Don't care, throw it away
		}
		writeWaypoint(Lat, Lon, bonus, title)
	}
	fmt.Printf("%v bonuses read including %v with bad coordinates\n\n", bonusCount, badCoordCount)

}
