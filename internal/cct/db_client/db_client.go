package db_client

//go:generate mockgen -destination=./db_client_mock.go -package=db_client -source=db_client.go

import (
	"log"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
)

// UsageData TODO
type UsageData struct {
	Cost     float64
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

// Connection TODO
type conClient interface {
	Write(bp client.BatchPoints) error
	Close() error
	Ping(timeout time.Duration) (time.Duration, string, error)
	Query(q client.Query) (*client.Response, error)
}

// BP TODO
type bp interface {
	AddPoint(p *client.Point)
	AddPoints(ps []*client.Point)
	Points() []*client.Point
	Precision() string
	SetPrecision(s string) error
	Database() string
	SetDatabase(s string)
	WriteConsistency() string
	SetWriteConsistency(s string)
	RetentionPolicy() string
	SetRetentionPolicy(s string)
}

// HTTPClient TODO
type httpClient interface {
	NewHTTPClient(conf client.HTTPConfig) (client.Client, error)
}

// BatchPoints TODO
type batchPoints interface {
	NewBatchPoints(conf client.BatchPointsConfig) (client.BatchPoints, error)
}

// Point TODO
type point interface {
	NewPoint(name string, tags map[string]string, fields map[string]interface{}, t ...time.Time) (*client.Point, error)
}

// DBClient Can be used to add UsageData to a DB
type DBClient struct {
	config DBClientConfig
	httpClient
	batchPoints
	point
}

type httpClientStruct struct{}
type batchPointsStruct struct{}
type pointsStruct struct{}

// NewHTTPClient TODO
func (e httpClientStruct) NewHTTPClient(conf client.HTTPConfig) (client.Client, error) {
	return client.NewHTTPClient(conf)
}

// NewBatchPoints TODO
func (e batchPointsStruct) NewBatchPoints(conf client.BatchPointsConfig) (client.BatchPoints, error) {
	return client.NewBatchPoints(conf)
}

// NewPoint TODO
func (e pointsStruct) NewPoint(name string, tags map[string]string, fields map[string]interface{}, t ...time.Time) (*client.Point, error) {
	return client.NewPoint(name, tags, fields, t...)
}

// NewDBClient initializes a DBClient
func NewDBClient(config DBClientConfig) DBClient {
	return DBClient{
		config:      config,
		httpClient:  httpClientStruct{},
		batchPoints: batchPointsStruct{},
		point:       pointsStruct{},
	}
}

// GetConfig TODO
func (e *DBClient) GetConfig() DBClientConfig {
	return e.config
}

// AddUsageData Adds an array of UsageData to the DB
func (e *DBClient) AddUsageData(usageData []UsageData) bool {
	var c conClient
	c, err := e.httpClient.NewHTTPClient(client.HTTPConfig{
		Addr:     e.config.Address,
		Username: e.config.Username,
		Password: e.config.Password,
	})
	if err != nil {
		log.Println(err)
		return false
	}

	defer c.Close()

	for _, data := range usageData {
		bp, err := e.createBatchPoints(data)
		if err != nil {
			log.Println(err)
			return false
		}

		if err := c.Write(bp); err != nil {
			log.Fatal(err)
		}
	}

	if err := c.Close(); err != nil {
		log.Fatal(err)
	}

	return true
}

// createBatchPoints Creates a batch of points from one UsageData that can be added to the DB
// If the second parameter is nil no error occurred.
func (e *DBClient) createBatchPoints(data UsageData) (bp, error) {
	// Create a new point batch
	var bp bp
	bp, err := e.batchPoints.NewBatchPoints(client.BatchPointsConfig{
		Database:  e.config.DBName,
		Precision: "h",
	})
	if err != nil {
		return nil, err
	}

	// Convert decimal to float and add as field
	cost := data.Cost
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
	pt, err := e.point.NewPoint("cost", m, fields, data.Date)
	if err != nil {
		return nil, err
	}
	bp.AddPoint(pt)

	return bp, nil
}
