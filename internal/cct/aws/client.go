// Package aws is responsible for fetching information about aws billing
package aws

import (
	"encoding/csv"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lentzi90/cloud-cost-tracker/internal/cct/dbclient"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

// Client represents a connection to an aws S3 bucket
type Client struct {
	bucket  string
	service *s3.S3
}

// NewClient initializes a new S3 connection
func NewClient(bucket string) Client {
	svc := newS3Service()
	client := Client{bucket: bucket, service: svc}
	return client
}

func newS3Service() *s3.S3 {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return s3.New(sess)
}

func (client *Client) getTable(key string, query string) ([][]string, error) {

	// Select contents of target csv
	params := &s3.SelectObjectContentInput{
		Bucket:         aws.String(client.bucket),
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
	resp, err := client.service.SelectObjectContent(params)
	if err != nil {
		return nil, err
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

	return tbl, nil
}

func overlap(a int64, b int64, c int64, d int64) float64 {
	return math.Max(0, math.Min(float64(b), float64(d))-math.Max(float64(a), float64(c)))
}

func calculateRatio(start time.Time, stop time.Time, date time.Time) float64 {
	newDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	nominator := overlap(start.Unix(), stop.Unix(), newDate.Unix(), newDate.AddDate(0, 0, 1).Unix())
	denominator := float64(stop.Unix() - start.Unix())
	res := nominator / denominator
	return res
}

// GetCloudCost returns information about the cost during a specific day
func (client *Client) GetCloudCost(timestamp time.Time) ([]dbclient.UsageData, error) {
	// S3 Select
	// 0  1         2        3       4        5
	// id StartDate StopDate Service Currency BlendedCost
	query := "SELECT s._1, s._11, s._12, s._13, s._20, s._24 FROM S3Object s"

	form := "2006-01-02T15:04:05Z"

	// Calculate the correct key
	key := client.getBucketKey(timestamp)

	// Get table from bucket with key using query
	tbl, err := client.getTable(key, query)
	if err != nil {
		return nil, err
	}

	// Transform result into internal format []UsageData
	res := make([]dbclient.UsageData, 0)
	for _, val := range tbl {
		labels := map[string]string{}
		labels["id"] = val[0]
		labels["service"] = val[3]
		labels["currency"] = val[4]
		labels["cloud"] = "aws"
		start, _ := time.Parse(form, val[1])
		stop, _ := time.Parse(form, val[2])
		ratio := calculateRatio(start, stop, timestamp)
		cost, _ := strconv.ParseFloat(val[5], 64)
		row := dbclient.UsageData{
			Cost:   cost * ratio,
			Date:   timestamp,
			Labels: labels}
		res = append(res, row)
	}

	// Group similar UsageData
	res = groupUsageData(res)

	return res, nil
}

// TODO
func groupUsageData(data []dbclient.UsageData) []dbclient.UsageData {
	return data
}

func keyIsValid(key string, date string) bool {
	if !strings.Contains(key, date) {
		return false
	} else if !strings.Contains(key, "csv.gz") {
		return false
	}
	return true
}

func selectKey(objects []*s3.Object, date string) string {
	var latest time.Time
	var key string
	latest = *objects[0].LastModified
	key = *objects[0].Key
	for _, val := range objects {
		if keyIsValid(*val.Key, date) {
			if val.LastModified.After(latest) {
				latest = *val.LastModified
				key = *val.Key
			}
		}
	}
	return key
}

func (client *Client) getBucketKey(timestamp time.Time) string {
	params := &s3.ListObjectsInput{
		Bucket: aws.String(client.bucket),
		Prefix: aws.String("daily-report/test-usage-report"),
	}

	form := "20060102"
	start := time.Date(timestamp.Year(), timestamp.Month(), 1, 0, 0, 0, 0, timestamp.Location())
	stop := start.AddDate(0, 1, 0)
	date := start.Format(form) + "-" + stop.Format(form)

	resp, _ := client.service.ListObjects(params)
	key := selectKey(resp.Contents, date)

	return key
}
