// Package aws ...
package aws

import (
	"encoding/csv"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
	"io"
	"log"
	"math"
	"strconv"
	"time"
)

// Client ...
type Client struct {
	bucket string
	key    string
}

// NewClient ...
func NewClient(bucket string, key string) Client {
	client := Client{bucket: bucket, key: key}
	return client
}

func newS3Service() *s3.S3 {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return s3.New(sess)
}

func getTable(bucket string, key string, query string) [][]string {

	svc := newS3Service()

	// Select contents of target csv
	params := &s3.SelectObjectContentInput{
		Bucket:         aws.String(bucket),
		Key:            aws.String(key),
		ExpressionType: aws.String(s3.ExpressionTypeSql),
		Expression:     aws.String(query),
		InputSerialization: &s3.InputSerialization{
			CompressionType: aws.String("gzip"),
			CSV: &s3.CSVInput{
				FileHeaderInfo: aws.String(s3.FileHeaderInfoIgnore),
			},
		},
		OutputSerialization: &s3.OutputSerialization{
			CSV: &s3.CSVOutput{},
		},
	}

	// Request stream
	resp, err := svc.SelectObjectContent(params)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.EventStream.Close()

	// Get data from stream
	results, resultWriter := io.Pipe()
	go func() {
		defer resultWriter.Close()
		for event := range resp.EventStream.Events() {
			switch e := event.(type) {
			case *s3.RecordsEvent:
				resultWriter.Write(e.Payload)
			}
		}
	}()

	// Read the csv into a slice
	tbl := make([][]string, 0)
	resReader := csv.NewReader(results)
	for {
		record, err := resReader.Read()
		if err == io.EOF {
			break
		}
		tbl = append(tbl, record)
	}

	return tbl
}

func s3ToUnix(timestamp string, form string) int {
	tmp, err := time.Parse(form, timestamp)
	if err != nil {
		log.Fatal(err)
	}
	unix := strconv.FormatInt(tmp.Unix(), 10)
	unixI, err := strconv.Atoi(unix)
	if err != nil {
		log.Fatal(err)
	}
	return unixI
}

func calculateOverlap(a int, b int, x int) float64 {
	y := (float64(x) - float64(a)) / (float64(b) - float64(a))
	y = math.Max(y, 0.0)
	y = math.Min(y, 1.0)
	return y
}

// GetAccumulatedCost ...
func GetAccumulatedCost(bucket string, key string, timestamp time.Time) []interface{} {

	// Query to be used by S3 Select
	query := "SELECT s._11, s._12, s._13, s._24 FROM S3Object s"

	// Get table from S3
	tbl := getTable(bucket, key, query)

	// Define S3 timestamp format using Go reference time
	form := "2006-01-02T15:04:05Z"
	baz := strconv.FormatInt(timestamp.Unix(), 10)

	// Transform
	for _, val := range tbl {
		a := s3ToUnix(val[0], form)
		b := s3ToUnix(val[1], form)
		x, _ := strconv.Atoi(baz)
		y := calculateOverlap(a, b, x)
		tmp, _ := strconv.ParseFloat(val[3], 64)
		val[3] = strconv.FormatFloat(tmp*y, 'f', -1, 64)
	}

	// Group by service
	m := map[string]float64{}
	for _, val := range tbl {
		res, _ := strconv.ParseFloat(val[3], 64)
		m[val[2]] += res
	}

	// Convert back to slice
	res := make([]interface{}, 0)
	for key, cost := range m {
		res = append(res, key)
		res = append(res, cost)
	}

	return res
}

func calculateRatio() {
}

// GetCloudCost ...
func (client *Client) GetCloudCost(timestamp time.Time) []dbclient.UsageData {
	// S3 Select query
	// StartDate StopDate Service Currency BlendedCost
	query := "SELECT s._11, s._12, s._13, s._20, s._24 FROM S3Object s"

	// Get table from bucket with key using query
	tbl := getTable(client.bucket, client.key, query)

	// Transform result into internal format []UsageData
	res := make([]dbclient.UsageData, 0)
	for _, val := range tbl {
		_ = val
		labels := map[string]string{}
		labels["Service"] = val[2]
		labels["Currency"] = val[3]
		row := dbclient.UsageData{
			Cost:   0.0,
			Date:   timestamp,
			Labels: labels}
		res = append(res, row)
	}

	return res
}
