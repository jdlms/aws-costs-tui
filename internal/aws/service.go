// service.go - service cost data fetching
package aws

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func getServiceData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set time range for last 30 days - exact logic from main.go
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	// Format dates as YYYY-MM-DD
	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	// Get cost and usage data, grouped by service - exact API call from main.go
	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
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

	rows := [][]string{
		{"Service", "Cost (30 days)", "Percentage"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return CostData{Title: "Costs by Service", Rows: rows}
	}

	var totalCost float64
	serviceMap := make(map[string]float64)

	// Aggregate costs across all time periods by service name
	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && group.Metrics != nil {
				serviceName := group.Keys[0]
				if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil && amount > 0 {
						serviceMap[serviceName] += amount
						totalCost += amount
					}
				}
			}
		}
	}

	// Convert map to slice for sorting
	var costGroups []CostGroup
	for serviceName, amount := range serviceMap {
		costGroups = append(costGroups, CostGroup{
			Name:   serviceName,
			Amount: amount,
		})
	}

	// Sort by amount (highest to lowest)
	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	for _, group := range costGroups {
		percentage := (group.Amount / totalCost) * 100
		amountStr := strconv.FormatFloat(group.Amount, 'f', -1, 64)
		rows = append(rows, []string{
			group.Name,
			formatCost(&amountStr),
			fmt.Sprintf("%.1f%%", percentage),
		})
	}

	return CostData{Title: "🛠️ Last 30 Days Costs by Service", Rows: rows}
}
