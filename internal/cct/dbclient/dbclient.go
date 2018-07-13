package dbclient

//go:generate mockgen -destination=./dbclient_mock.go -package=dbclient -source=dbclient.go

import (
	"time"

	client "github.com/influxdata/influxdb/client/v2"
)

// UsageData Struct that the submodules should return
type UsageData struct {
	Cost   float64
	Date   time.Time
	Labels map[string]string
}

// CloudCostClient The interface that all the cloudClients should implement
type CloudCostClient interface {
	GetCloudCost(time.Time) ([]UsageData, error)
}

// Config struct with connection information of the influxDB
type Config struct {
	DBName   string
	Username string
	Password string
	Address  string
}

// conClient Interface thats the same as client.Client to make testing easier
type conClient interface {
	Write(bp client.BatchPoints) error
	Close() error
	Ping(timeout time.Duration) (time.Duration, string, error)
	Query(q client.Query) (*client.Response, error)
}

// bp Interface thats the same as client.BatchPoints to make testing easier
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

// HTTPClient Interface with the functions that this package uses from client to make testing easier
type influxInterface interface {
	NewHTTPClient(conf client.HTTPConfig) (client.Client, error)
	NewBatchPoints(conf client.BatchPointsConfig) (client.BatchPoints, error)
	NewPoint(name string, tags map[string]string, fields map[string]interface{}, t ...time.Time) (*client.Point, error)
}

// influxClient Struct that will include the functions from influxInterface
type influxClient struct{}

// NewHTTPClient Proxy function to client
func (e influxClient) NewHTTPClient(conf client.HTTPConfig) (client.Client, error) {
	return client.NewHTTPClient(conf)
}

// NewBatchPoints Proxy function to client
func (e influxClient) NewBatchPoints(conf client.BatchPointsConfig) (client.BatchPoints, error) {
	return client.NewBatchPoints(conf)
}

// NewPoint Proxy function to client
func (e influxClient) NewPoint(name string, tags map[string]string, fields map[string]interface{}, t ...time.Time) (*client.Point, error) {
	return client.NewPoint(name, tags, fields, t...)
}

// DBClient Can be used to add UsageData to a DB
type DBClient struct {
	config Config
	influxInterface
}

// NewDBClient initializes a DBClient
func NewDBClient(config Config) DBClient {
	return DBClient{
		config:          config,
		influxInterface: influxClient{},
	}
}

// GetConfig Returns the config
func (e *DBClient) GetConfig() Config {
	return e.config
}

// AddUsageData Adds an array of UsageData to the DB
func (e *DBClient) AddUsageData(usageData []UsageData) error {
	var c conClient
	c, err := e.influxInterface.NewHTTPClient(client.HTTPConfig{
		Addr:     e.config.Address,
		Username: e.config.Username,
		Password: e.config.Password,
	})
	if err != nil {
		return err
	}

	defer c.Close()

	for _, data := range usageData {
		bp, err := e.createBatchPoints(data)
		if err != nil {
			return err
		}

		if err := c.Write(bp); err != nil {
			return err
		}
	}

	if err := c.Close(); err != nil {
		return err
	}

	return nil
}

// createBatchPoints Creates a batch of points from one UsageData that can be added to the DB
// If the second parameter is nil no error occurred.
func (e *DBClient) createBatchPoints(data UsageData) (bp, error) {
	// Create a new point batch
	var bp bp
	bp, err := e.influxInterface.NewBatchPoints(client.BatchPointsConfig{
		Database:  e.config.DBName,
		Precision: "h",
	})
	if err != nil {
		return nil, err
	}

	// Convert decimal to float and add as field
	cost := map[string]interface{}{"cost": data.Cost}

	// Create and add point
	pt, err := e.influxInterface.NewPoint("cost", data.Labels, cost, data.Date)
	if err != nil {
		return nil, err
	}
	bp.AddPoint(pt)

	return bp, nil
}
