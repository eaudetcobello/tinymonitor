// Package service enables the HTTP server to query data.
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eaudetcobello/tinymonitor/internal/model"
)

const timestampFormat = "2006-01-02T15:04:05.000000000Z"

type Metrics struct {
	DBClient *dynamodb.Client
}

type metricItem struct {
	AgentID uint32             `dynamodbav:"agent_id"`
	SK      string             `dynamodbav:"sk"` // "<kind>#<timestamp>"
	Kind    string             `dynamodbav:"kind"`
	Time    string             `dynamodbav:"time"`
	Values  map[string]float64 `dynamodbav:"values"`
	TTL     int64              `dynamodbav:"ttl"`
}

func toItem(m model.Metric, retention time.Duration) metricItem {
	ts := m.Time.UTC().Format(timestampFormat)

	return metricItem{
		AgentID: m.AgentID,
		SK:      string(m.Kind) + "#" + ts,
		Kind:    string(m.Kind),
		Time:    ts,
		Values:  m.Values,
		TTL:     m.Time.Add(retention).Unix(),
	}
}

func (s *Metrics) CreateTable(ctx context.Context) {
	_, err := s.DBClient.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:   aws.String("metrics"),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("agent_id"), AttributeType: types.ScalarAttributeTypeN},
			{AttributeName: aws.String("sk"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("agent_id"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("sk"), KeyType: types.KeyTypeRange},
		},
	})
	if err != nil {
		if _, ok := errors.AsType[*types.ResourceInUseException](err); !ok {
			log.Printf("create table: %v\n", err)
		}
	}

	err = dynamodb.NewTableExistsWaiter(s.DBClient).Wait(
		ctx,
		&dynamodb.DescribeTableInput{TableName: aws.String("metrics")},
		30*time.Second,
	)
	if err != nil {
		log.Printf("wait table exists: %v\n", err)
	}
}

func (s *Metrics) Get(ctx context.Context, agentID uint32, from time.Time, to time.Time) ([]model.Metric, error) {
	return nil, nil
}

func (s *Metrics) Put(ctx context.Context, metric model.Metric) error {
	metricItem, err := attributevalue.MarshalMap(toItem(metric, 24*7*52*time.Hour))
	if err != nil {
		return fmt.Errorf("marshal map: %w", err)
	}

	_, err = s.DBClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("metrics"),
		Item:      metricItem,
	})
	if err != nil {
		return fmt.Errorf("put metric: %w", err)
	}

	return nil
}
