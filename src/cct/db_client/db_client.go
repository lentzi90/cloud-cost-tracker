package db_client

import (
	"log"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
	"github.com/shopspring/decimal"
)

// UsageData TODO
type UsageData struct {
	Cost     decimal.Decimal
	Currency string
	Date     time.Time
	Labels   map[string]string
}

// DBClientConfig Config struct with connection information of the influxDB
type DBClientConfig struct {
	DBName   string
	Username string
	Password string
	Address  string
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
		Addr:     e.config.Address,
		Username: e.config.Username,
		Password: e.config.Password,
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
	cost, _ := data.Cost.Float64()
	fields := map[string]interface{}{
		"cost": cost,
		//"currency": data.currency, //If the currency should be a value
	}

	// Merge currency into label map
	m := map[string]string{}

	if data.Labels != nil {
		data.Labels["currency"] = data.Currency
		m = data.Labels
	} else {
		m["currency"] = data.Currency
	}

	// Create and add point
	pt, err := client.NewPoint("cost", m, fields, data.Date)
	if err != nil {
		return nil, err
	}
	bp.AddPoint(pt)

	return bp, nil
}
