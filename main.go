package main

import (
	"context"
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

	monitor(ctx, collector.Gopsutil{}, metricsService)
}

func monitor(ctx context.Context, c collector.Collector, metricsService *service.Metrics) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping...")

			return
		case <-ticker.C:
			metric, err := c.Memory()
			if err != nil {
				log.Printf("collect memory: %v", err)

				continue
			}

			if err := metricsService.Put(ctx, metric); err != nil {
				log.Printf("store metric: %v", err)

				continue
			}

			log.Printf("put metric")
		}
	}
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
