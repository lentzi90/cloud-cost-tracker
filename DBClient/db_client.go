package main

import (
	"log"

	client "github.com/influxdata/influxdb/client/v2"
)

// UsageData TODO
/*type UsageData struct {
	cost     decimal.Decimal
	currency string
	date     time.Time
	labels   map[string]string
}*/

// DBClientConfig Config struct with connection information of the influxDB
type DBClientConfig struct {
	DBName   string
	username string
	password string
	address  string
}

// DBClient Can be used to add UsageData to a DB
type DBClient struct {
	config DBClientConfig
}

// NewDBClient initializes a DBClient
func NewDBClient(config DBClientConfig) DBClient {
	return DBClient{config: config}
}

// AddUsageData Adds an array of UsageData to the DB
func (e *DBClient) AddUsageData(usageData []UsageData) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     e.config.address,
		Username: e.config.username,
		Password: e.config.password,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	for _, data := range usageData {
		bp, err := e.CreateBatchPoints(data)
		if err != nil {
			log.Fatal(err)
			break
		}

		if err := c.Write(bp); err != nil {
			log.Fatal(err)
		}
	}

	if err := c.Close(); err != nil {
		log.Fatal(err)
	}
}

// CreateBatchPoints Creates a batch of points from one UsageData that can be added to the DB
// If the second parameter is nil no error occurred.
func (e *DBClient) CreateBatchPoints(data UsageData) (client.BatchPoints, error) {
	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  e.config.DBName,
		Precision: "h",
	})
	if err != nil {
		return nil, err
	}

	// Convert decimal to float and add as field
	cost, _ := data.cost.Float64()
	fields := map[string]interface{}{
		"cost": cost,
		//"currency": data.currency, //If the currency should be a value
	}

	// Merge currency into label map
	m := map[string]string{}

	if data.labels != nil {
		data.labels["currency"] = data.currency
		m = data.labels
	} else {
		m["currency"] = data.currency
	}

	// Create and add point
	pt, err := client.NewPoint("cost", m, fields, data.date)
	if err != nil {
		return nil, err
	}
	bp.AddPoint(pt)

	return bp, nil
}
