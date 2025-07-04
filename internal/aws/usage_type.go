// usage_type.go - usage type cost data fetching
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

func getUsageTypeData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	// Get costs by usage type - exact API call from main.go
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
				Key:  &[]string{"USAGE_TYPE"}[0],
			},
		},
	})

	rows := [][]string{
		{"Usage Type", "Cost (30 days)", "Percentage"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return CostData{Title: "Costs by Usage Type", Rows: rows}
	}

	var totalCost float64
	usageTypeMap := make(map[string]float64)

	// Aggregate costs across all time periods by usage type
	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && group.Metrics != nil {
				usageTypeName := group.Keys[0]
				if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil && amount > 0 {
						usageTypeMap[usageTypeName] += amount
						totalCost += amount
					}
				}
			}
		}
	}

	// Convert map to slice for sorting
	var costGroups []CostGroup
	for usageTypeName, amount := range usageTypeMap {
		costGroups = append(costGroups, CostGroup{
			Name:   usageTypeName,
			Amount: amount,
		})
	}

	// Sort by amount (highest to lowest)
	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	// Take only top 10
	if len(costGroups) > 10 {
		costGroups = costGroups[:10]
		// Recalculate total for percentage calculation
		totalCost = 0
		for _, group := range costGroups {
			totalCost += group.Amount
		}
	}

	for _, group := range costGroups {
		percentage := (group.Amount / totalCost) * 100
		amountStr := strconv.FormatFloat(group.Amount, 'f', -1, 64)
		rows = append(rows, []string{
			group.Name,
			formatCost(&amountStr),
			fmt.Sprintf("%.1f%%", percentage),
		})
	}

	return CostData{Title: "📊 Top 10 Usage Types", Rows: rows}
}
