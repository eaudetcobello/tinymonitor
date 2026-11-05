package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/eaudetcobello/tinymonitor/internal/service"
)

const usedPct = "used_pct"

var errNoMorePages = errors.New("no more pages")

// fakeDB serves Query from pre-built pages and records every Query input.
// Other DynamoDB methods are left nil and panic if called.
type fakeDB struct {
	service.DynamoDB

	pages   []*dynamodb.QueryOutput
	queries []*dynamodb.QueryInput
}

func (f *fakeDB) Query(
	_ context.Context, in *dynamodb.QueryInput, _ ...func(*dynamodb.Options),
) (*dynamodb.QueryOutput, error) {
	f.queries = append(f.queries, in)
	if len(f.queries) > len(f.pages) {
		return nil, errNoMorePages
	}

	return f.pages[len(f.queries)-1], nil
}

func item(ts, used string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"agent_id": &types.AttributeValueMemberN{Value: "7"},
		"sk":       &types.AttributeValueMemberS{Value: "memory#" + ts},
		"kind":     &types.AttributeValueMemberS{Value: "memory"},
		"time":     &types.AttributeValueMemberS{Value: ts},
		"values": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			usedPct: &types.AttributeValueMemberN{Value: used},
		}},
	}
}

func stringValue(t *testing.T, av types.AttributeValue) string {
	t.Helper()

	switch v := av.(type) {
	case *types.AttributeValueMemberS:
		return v.Value
	case *types.AttributeValueMemberN:
		return v.Value
	default:
		t.Fatalf("unexpected attribute value type %T", av)

		return ""
	}
}

func twoPageDB() (*fakeDB, map[string]types.AttributeValue) {
	lastKey := map[string]types.AttributeValue{
		"agent_id": &types.AttributeValueMemberN{Value: "7"},
		"sk":       &types.AttributeValueMemberS{Value: "memory#2026-11-04T10:00:01.000000000Z"},
	}

	return &fakeDB{pages: []*dynamodb.QueryOutput{
		{
			Items: []map[string]types.AttributeValue{
				item("2026-11-04T10:00:00.000000000Z", "1"),
				item("2026-11-04T10:00:01.000000000Z", "2"),
			},
			LastEvaluatedKey: lastKey,
		},
		{
			Items: []map[string]types.AttributeValue{
				item("2026-11-04T10:00:02.500000000Z", "3"),
			},
		},
	}}, lastKey
}

func TestGetQueriesSortKeyRange(t *testing.T) {
	t.Parallel()

	db, _ := twoPageDB()
	svc := &service.Metrics{DBClient: db}

	// A non-UTC zone checks that bounds are normalized to UTC before building sort keys.
	zone := time.FixedZone("EST", -5*3600)
	from := time.Date(2026, 11, 4, 5, 0, 0, 0, zone)
	to := time.Date(2026, 11, 4, 5, 1, 0, 0, zone)

	if _, err := svc.Get(t.Context(), 7, model.KindMemory, from, to); err != nil {
		t.Fatalf("Get: %v", err)
	}

	q := db.queries[0]
	if want := "agent_id = :agent_id AND sk BETWEEN :from AND :to"; aws.ToString(q.KeyConditionExpression) != want {
		t.Errorf("key condition = %q, want %q", aws.ToString(q.KeyConditionExpression), want)
	}

	if !aws.ToBool(q.ScanIndexForward) {
		t.Error("ScanIndexForward is false, want ascending order")
	}

	wantValues := map[string]string{
		":agent_id": "7",
		":from":     "memory#2026-11-04T10:00:00.000000000Z",
		":to":       "memory#2026-11-04T10:01:00.000000000Z",
	}
	for name, want := range wantValues {
		if got := stringValue(t, q.ExpressionAttributeValues[name]); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestGetFollowsPages(t *testing.T) {
	t.Parallel()

	db, lastKey := twoPageDB()
	svc := &service.Metrics{DBClient: db}

	got, err := svc.Get(t.Context(), 7, model.KindMemory, time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if len(db.queries) != 2 {
		t.Fatalf("got %d queries, want 2", len(db.queries))
	}

	if db.queries[0].ExclusiveStartKey != nil {
		t.Error("first query has an ExclusiveStartKey")
	}

	if !reflect.DeepEqual(db.queries[1].ExclusiveStartKey, lastKey) {
		t.Errorf("second query ExclusiveStartKey = %v, want %v", db.queries[1].ExclusiveStartKey, lastKey)
	}

	base := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)

	want := []model.Metric{
		{AgentID: 7, Kind: model.KindMemory, Time: base, Values: map[string]float64{usedPct: 1}},
		{AgentID: 7, Kind: model.KindMemory, Time: base.Add(time.Second), Values: map[string]float64{usedPct: 2}},
		{AgentID: 7, Kind: model.KindMemory, Time: base.Add(2500 * time.Millisecond), Values: map[string]float64{usedPct: 3}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v, want %+v", got, want)
	}
}

func TestGetEmptyRangeReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	svc := &service.Metrics{DBClient: &fakeDB{pages: []*dynamodb.QueryOutput{{}}}}

	now := time.Now()

	got, err := svc.Get(t.Context(), 7, model.KindCPU, now.Add(-time.Minute), now)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got == nil || len(got) != 0 {
		t.Errorf("Get = %#v, want non-nil empty slice", got)
	}
}

func TestGetReturnsQueryError(t *testing.T) {
	t.Parallel()

	svc := &service.Metrics{DBClient: &fakeDB{}}

	now := time.Now()

	_, err := svc.Get(t.Context(), 7, model.KindCPU, now.Add(-time.Minute), now)
	if !errors.Is(err, errNoMorePages) {
		t.Errorf("Get error = %v, want %v", err, errNoMorePages)
	}
}
