// Package service enables the HTTP server to query data.
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eaudetcobello/tinymonitor/internal/model"
)

const (
	tableName = "metrics"
	// timestampFormat has a fixed width, so sort keys order lexically by time.
	timestampFormat = "2006-01-02T15:04:05.000000000Z"
)

// DynamoDB is the subset of the DynamoDB API used by Metrics. *dynamodb.Client implements it.
type DynamoDB interface {
	CreateTable(
		ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options),
	) (*dynamodb.CreateTableOutput, error)
	DescribeTable(
		ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options),
	) (*dynamodb.DescribeTableOutput, error)
	PutItem(
		ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options),
	) (*dynamodb.PutItemOutput, error)
	Query(
		ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options),
	) (*dynamodb.QueryOutput, error)
}

type Metrics struct {
	DBClient DynamoDB
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
	return metricItem{
		AgentID: m.AgentID,
		SK:      sortKey(m.Kind, m.Time),
		Kind:    string(m.Kind),
		Time:    m.Time.UTC().Format(timestampFormat),
		Values:  m.Values,
		TTL:     m.Time.Add(retention).Unix(),
	}
}

func sortKey(kind model.Kind, t time.Time) string {
	return string(kind) + "#" + t.UTC().Format(timestampFormat)
}

func (i metricItem) toMetric() (model.Metric, error) {
	t, err := time.Parse(timestampFormat, i.Time)
	if err != nil {
		return model.Metric{}, fmt.Errorf("parse time %q: %w", i.Time, err)
	}

	return model.Metric{
		AgentID: i.AgentID,
		Kind:    model.Kind(i.Kind),
		Time:    t,
		Values:  i.Values,
	}, nil
}

func (s *Metrics) CreateTable(ctx context.Context) {
	_, err := s.DBClient.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:   aws.String(tableName),
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
		&dynamodb.DescribeTableInput{TableName: aws.String(tableName)},
		30*time.Second,
	)
	if err != nil {
		log.Printf("wait table exists: %v\n", err)
	}
}

// Get returns the metrics of one kind for an agent with a time in [from, to], ordered by time ascending.
func (s *Metrics) Get(
	ctx context.Context, agentID uint32, kind model.Kind, from, to time.Time,
) ([]model.Metric, error) {
	paginator := dynamodb.NewQueryPaginator(s.DBClient, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("agent_id = :agent_id AND sk BETWEEN :from AND :to"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":agent_id": &types.AttributeValueMemberN{Value: strconv.FormatUint(uint64(agentID), 10)},
			":from":     &types.AttributeValueMemberS{Value: sortKey(kind, from)},
			":to":       &types.AttributeValueMemberS{Value: sortKey(kind, to)},
		},
		ScanIndexForward: aws.Bool(true),
	})

	metrics := make([]model.Metric, 0)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("query metrics: %w", err)
		}

		var items []metricItem
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &items); err != nil {
			return nil, fmt.Errorf("unmarshal metrics: %w", err)
		}

		for _, item := range items {
			m, err := item.toMetric()
			if err != nil {
				return nil, err
			}

			metrics = append(metrics, m)
		}
	}

	return metrics, nil
}

func (s *Metrics) Put(ctx context.Context, metric model.Metric) error {
	metricItem, err := attributevalue.MarshalMap(toItem(metric, 24*7*52*time.Hour))
	if err != nil {
		return fmt.Errorf("marshal map: %w", err)
	}

	_, err = s.DBClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      metricItem,
	})
	if err != nil {
		return fmt.Errorf("put metric: %w", err)
	}

	return nil
}
