package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func main() {
	// Load AWS configuration from ~/.aws automatically
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Create Cost Explorer client
	client := costexplorer.NewFromConfig(cfg)

	// Set time range for last 30 days
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	// Format dates as YYYY-MM-DD
	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	// Get cost and usage data, grouped by service
	result, err := client.GetCostAndUsage(context.TODO(), &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  &[]string{"SERVICE"}[0],
			},
		},
	})

	if err != nil {
		log.Fatalf("Failed to get cost data: %v", err)
	}

	// Display cost per service
	fmt.Println("AWS service costs for the last 30 days:")
	for _, resultByTime := range result.ResultsByTime {
		fmt.Printf("Period: %s to %s\n", *resultByTime.TimePeriod.Start, *resultByTime.TimePeriod.End)
		for _, group := range resultByTime.Groups {
			service := "Unknown"
			if len(group.Keys) > 0 {
				service = group.Keys[0]
			}
			amount := group.Metrics["BlendedCost"].Amount
			unit := group.Metrics["BlendedCost"].Unit
			fmt.Printf("  %s: %s %s\n", service, *amount, *unit)
		}
		fmt.Println()
	}
}
