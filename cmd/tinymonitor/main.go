package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/eaudetcobello/tinymonitor/internal/collector"
	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/eaudetcobello/tinymonitor/internal/server"
	"github.com/eaudetcobello/tinymonitor/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	metricsService := &service.Metrics{
		DBClient: createDynamoClient(ctx),
	}

	metricsService.CreateTable(ctx)

	go server.Start(ctx, metricsService)

	monitor(ctx, &collector.Gopsutil{}, metricsService)
}

// monitor collects and stores a memory and a CPU sample every tick until ctx ends.
func monitor(ctx context.Context, c collector.Collector, metricsService *service.Metrics) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping...")

			return
		case <-ticker.C:
			collect(ctx, "memory", c.Memory, metricsService)
			collect(ctx, "cpu", c.CPU, metricsService)
		}
	}
}

// collect stores one sample from read. A failure is logged, except the CPU collector's first baseline reading.
func collect(ctx context.Context, name string, read func() (model.Metric, error), metricsService *service.Metrics) {
	metric, err := read()
	if errors.Is(err, collector.ErrNoBaseline) {
		return
	}

	if err != nil {
		log.Printf("collect %s: %v", name, err)

		return
	}

	if err := metricsService.Put(ctx, metric); err != nil {
		log.Printf("store %s metric: %v", name, err)

		return
	}

	log.Printf("put %s metric", name)
}

func createDynamoClient(ctx context.Context) *dynamodb.Client {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("local", "local", ""),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://localhost:8000")
	})

	result, err := client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		log.Fatalf("list tables: %v\n", err)
	}

	log.Printf("found tables: %v\n", result.TableNames)

	return client
}
