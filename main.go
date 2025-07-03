package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// Helper struct for sorting cost groups
type CostGroup struct {
	Name   string
	Amount float64
}

func main() {
	// Load AWS configuration from ~/.aws automatically
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Create Cost Explorer client
	client := costexplorer.NewFromConfig(cfg)

	// Track execution time
	start := time.Now()
	fmt.Println("Starting concurrent AWS Cost Explorer queries...")

	// Use WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	// Use a channel to control output ordering
	results := make(chan string, 4)

	// Start all API calls concurrently
	wg.Add(4)

	go func() {
		defer wg.Done()
		output := captureCurrentMonthCosts(client)
		results <- output
	}()

	go func() {
		defer wg.Done()
		output := captureForecastedCosts(client)
		results <- output
	}()

	go func() {
		defer wg.Done()
		output := captureAdditionalCostInsights(client)
		results <- output
	}()

	go func() {
		defer wg.Done()
		output := captureLast30DaysCostsByService(client)
		results <- output
	}()

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Print results as they come in
	for result := range results {
		fmt.Print(result)
	}

	elapsed := time.Since(start)
	fmt.Printf("All queries completed in %v (concurrent execution)\n", elapsed)
}

// Capture functions that return strings instead of printing directly
func captureCurrentMonthCosts(client *costexplorer.Client) string {
	var output strings.Builder

	now := time.Now()
	// Get first day of current month
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// Get first day of next month
	nextMonthStart := currentMonthStart.AddDate(0, 1, 0)

	startDate := currentMonthStart.Format("2006-01-02")
	endDate := nextMonthStart.Format("2006-01-02")

	output.WriteString(fmt.Sprintf("=== CURRENT MONTH COSTS (%s) ===\n", now.Format("January 2006")))

	result, err := client.GetCostAndUsage(context.TODO(), &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost", "UnblendedCost", "NetUnblendedCost"},
	})

	if err != nil {
		output.WriteString(fmt.Sprintf("Failed to get current month cost data: %v\n", err))
		return output.String()
	}

	for _, resultByTime := range result.ResultsByTime {
		output.WriteString(fmt.Sprintf("Period: %s to %s\n", *resultByTime.TimePeriod.Start, *resultByTime.TimePeriod.End))

		if blendedCost, exists := resultByTime.Total["BlendedCost"]; exists {
			output.WriteString(fmt.Sprintf("  Total Blended Cost: %s\n", formatCost(blendedCost.Amount)))
		}
		if unblendedCost, exists := resultByTime.Total["UnblendedCost"]; exists {
			output.WriteString(fmt.Sprintf("  Total Unblended Cost: %s\n", formatCost(unblendedCost.Amount)))
		}
		if netCost, exists := resultByTime.Total["NetUnblendedCost"]; exists {
			output.WriteString(fmt.Sprintf("  Total Net Cost: %s\n", formatCost(netCost.Amount)))
		}
	}
	output.WriteString("\n")
	return output.String()
}

func captureForecastedCosts(client *costexplorer.Client) string {
	var output strings.Builder

	now := time.Now()
	// Forecast for next month
	nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	if nextMonthStart.Month() == 1 { // Handle year rollover
		nextMonthStart = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
	}
	nextMonthEnd := nextMonthStart.AddDate(0, 1, 0)

	startDate := nextMonthStart.Format("2006-01-02")
	endDate := nextMonthEnd.Format("2006-01-02")

	output.WriteString(fmt.Sprintf("=== FORECASTED COSTS (%s) ===\n", nextMonthStart.Format("January 2006")))

	forecast, err := client.GetCostForecast(context.TODO(), &costexplorer.GetCostForecastInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metric:      types.MetricBlendedCost,
	})

	if err != nil {
		output.WriteString(fmt.Sprintf("Failed to get cost forecast: %v\n", err))
		return output.String()
	}

	output.WriteString(fmt.Sprintf("Forecast Period: %s to %s\n", startDate, endDate))
	output.WriteString(fmt.Sprintf("Predicted Total Cost: %s\n", formatCost(forecast.Total.Amount)))

	if forecast.ForecastResultsByTime != nil {
		for _, forecastResult := range forecast.ForecastResultsByTime {
			output.WriteString(fmt.Sprintf("  Period: %s to %s\n", *forecastResult.TimePeriod.Start, *forecastResult.TimePeriod.End))
			output.WriteString(fmt.Sprintf("  Mean Estimate: %s\n", formatCost(forecastResult.MeanValue)))
		}
	}
	output.WriteString("\n")
	return output.String()
}

func captureLast30DaysCostsByService(client *costexplorer.Client) string {
	var output strings.Builder

	// Set time range for last 30 days
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	// Format dates as YYYY-MM-DD
	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	output.WriteString("=== LAST 30 DAYS COSTS BY SERVICE ===\n")

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
		output.WriteString(fmt.Sprintf("Failed to get cost data: %v\n", err))
		return output.String()
	}

	// Display cost per service
	for _, resultByTime := range result.ResultsByTime {
		output.WriteString(fmt.Sprintf("Period: %s to %s\n", *resultByTime.TimePeriod.Start, *resultByTime.TimePeriod.End))
		sortedGroups := sortCostGroups(resultByTime.Groups)
		for _, group := range sortedGroups {
			if group.Amount > 0 { // Only show services with actual costs
				amountStr := strconv.FormatFloat(group.Amount, 'f', -1, 64)
				output.WriteString(fmt.Sprintf("  %s: %s\n", group.Name, formatCost(&amountStr)))
			}
		}
		output.WriteString("\n")
	}
	return output.String()
}

func captureAdditionalCostInsights(client *costexplorer.Client) string {
	var output strings.Builder

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	output.WriteString("=== ADDITIONAL COST INSIGHTS (Last 30 Days) ===\n")

	// Get costs by region
	output.WriteString("--- Costs by Region ---\n")
	regionResult, err := client.GetCostAndUsage(context.TODO(), &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  &[]string{"REGION"}[0],
			},
		},
	})

	if err == nil {
		for _, resultByTime := range regionResult.ResultsByTime {
			sortedGroups := sortCostGroups(resultByTime.Groups)
			for _, group := range sortedGroups {
				if group.Amount > 0 { // Only show regions with actual costs
					amountStr := strconv.FormatFloat(group.Amount, 'f', -1, 64)
					output.WriteString(fmt.Sprintf("  %s: %s\n", group.Name, formatCost(&amountStr)))
				}
			}
		}
	} else {
		output.WriteString(fmt.Sprintf("  Error getting region costs: %v\n", err))
	}

	// Get costs by usage type
	output.WriteString("\n--- Top Usage Types ---\n")
	usageResult, err := client.GetCostAndUsage(context.TODO(), &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  &[]string{"USAGE_TYPE"}[0],
			},
		},
	})

	if err == nil {
		for _, resultByTime := range usageResult.ResultsByTime {
			sortedGroups := sortCostGroups(resultByTime.Groups)
			count := 0
			for _, group := range sortedGroups {
				if count >= 10 { // Show top 10 usage types
					break
				}
				if group.Amount > 0 { // Only show usage types with actual costs
					amountStr := strconv.FormatFloat(group.Amount, 'f', -1, 64)
					output.WriteString(fmt.Sprintf("  %s: %s\n", group.Name, formatCost(&amountStr)))
					count++
				}
			}
		}
	} else {
		output.WriteString(fmt.Sprintf("  Error getting usage type costs: %v\n", err))
	}

	output.WriteString("\n")
	return output.String()
}

// formatCost converts AWS cost string to formatted dollars and cents, rounding up on 3rd decimal
func formatCost(amountStr *string) string {
	if amountStr == nil {
		return "$0.00"
	}

	amount, err := strconv.ParseFloat(*amountStr, 64)
	if err != nil {
		return "$0.00"
	}

	// Round up on 3rd decimal place (e.g., 4.445 becomes 4.45, 4.444 becomes 4.45)
	rounded := math.Ceil(amount*100) / 100

	return fmt.Sprintf("$%.2f", rounded)
}

// sortCostGroups sorts groups by cost amount (highest to lowest)
func sortCostGroups(groups []types.Group) []CostGroup {
	var costGroups []CostGroup

	for _, group := range groups {
		name := "Unknown"
		if len(group.Keys) > 0 {
			name = group.Keys[0]
		}

		amount := 0.0
		if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
			if parsed, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil {
				amount = parsed
			}
		}

		costGroups = append(costGroups, CostGroup{
			Name:   name,
			Amount: amount,
		})
	}

	// Sort by amount (highest to lowest)
	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	return costGroups
}
