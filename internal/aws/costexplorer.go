package aws

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	awstypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"cost-explorer/internal/types"
)

// GetDashboardData fetches dashboard overview data
func GetDashboardData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonthStart := currentMonthStart.AddDate(0, 1, 0)

	startDate := currentMonthStart.Format("2006-01-02")
	endDate := nextMonthStart.Format("2006-01-02")

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &awstypes.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"BlendedCost", "UnblendedCost", "NetUnblendedCost"},
	})

	rows := [][]string{
		{"Metric", "Amount", "Period"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return types.CostData{Title: "Dashboard Overview", Rows: rows}
	}

	for _, resultByTime := range result.ResultsByTime {
		period := fmt.Sprintf("%s to %s", *resultByTime.TimePeriod.Start, *resultByTime.TimePeriod.End)

		if blendedCost, exists := resultByTime.Total["BlendedCost"]; exists {
			rows = append(rows, []string{"Total Blended Cost", formatCost(blendedCost.Amount), period})
		}
		if unblendedCost, exists := resultByTime.Total["UnblendedCost"]; exists {
			rows = append(rows, []string{"Total Unblended Cost", formatCost(unblendedCost.Amount), period})
		}
		if netCost, exists := resultByTime.Total["NetUnblendedCost"]; exists {
			rows = append(rows, []string{"Total Net Cost", formatCost(netCost.Amount), period})
		}
	}

	return types.CostData{Title: "💸 Dashboard Overview", Rows: rows}
}

// GetForecastData fetches cost forecast data
func GetForecastData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	if nextMonthStart.Month() == 1 {
		nextMonthStart = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
	}
	nextMonthEnd := nextMonthStart.AddDate(0, 1, 0)

	startDate := nextMonthStart.Format("2006-01-02")
	endDate := nextMonthEnd.Format("2006-01-02")

	forecast, err := client.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
		TimePeriod: &awstypes.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: awstypes.GranularityMonthly,
		Metric:      awstypes.MetricBlendedCost,
	})

	rows := [][]string{
		{"Forecast Type", "Amount", "Period"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return types.CostData{Title: "Cost Forecast", Rows: rows}
	}

	period := fmt.Sprintf("%s to %s", startDate, endDate)
	rows = append(rows, []string{"Predicted Total Cost", formatCost(forecast.Total.Amount), period})

	if forecast.ForecastResultsByTime != nil {
		for _, forecastResult := range forecast.ForecastResultsByTime {
			forecastPeriod := fmt.Sprintf("%s to %s", *forecastResult.TimePeriod.Start, *forecastResult.TimePeriod.End)
			rows = append(rows, []string{"Mean Estimate", formatCost(forecastResult.MeanValue), forecastPeriod})
		}
	}

	return types.CostData{Title: "🔮 Cost Forecast", Rows: rows}
}

// GetServiceData fetches costs grouped by service
func GetServiceData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &awstypes.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []awstypes.GroupDefinition{
			{
				Type: awstypes.GroupDefinitionTypeDimension,
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
		return types.CostData{Title: "Costs by Service", Rows: rows}
	}

	var totalCost float64
	serviceMap := make(map[string]float64)

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

	var costGroups []types.CostGroup
	for serviceName, amount := range serviceMap {
		costGroups = append(costGroups, types.CostGroup{
			Name:   serviceName,
			Amount: amount,
		})
	}

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

	return types.CostData{Title: "🛠️ Last 30 Days Costs by Service", Rows: rows}
}

// GetRegionData fetches costs grouped by region
func GetRegionData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &awstypes.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []awstypes.GroupDefinition{
			{
				Type: awstypes.GroupDefinitionTypeDimension,
				Key:  &[]string{"REGION"}[0],
			},
		},
	})

	rows := [][]string{
		{"Region", "Cost (30 days)", "Percentage"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return types.CostData{Title: "Costs by Region", Rows: rows}
	}

	var totalCost float64
	regionMap := make(map[string]float64)

	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && group.Metrics != nil {
				regionName := group.Keys[0]
				if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil && amount > 0 {
						regionMap[regionName] += amount
						totalCost += amount
					}
				}
			}
		}
	}

	var costGroups []types.CostGroup
	for regionName, amount := range regionMap {
		costGroups = append(costGroups, types.CostGroup{
			Name:   regionName,
			Amount: amount,
		})
	}

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

	return types.CostData{Title: "🌍 Costs by Region", Rows: rows}
}

// GetUsageTypeData fetches costs grouped by usage type
func GetUsageTypeData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	startDate := thirtyDaysAgo.Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &awstypes.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"BlendedCost"},
		GroupBy: []awstypes.GroupDefinition{
			{
				Type: awstypes.GroupDefinitionTypeDimension,
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
		return types.CostData{Title: "Costs by Usage Type", Rows: rows}
	}

	var totalCost float64
	usageTypeMap := make(map[string]float64)

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

	var costGroups []types.CostGroup
	for usageTypeName, amount := range usageTypeMap {
		costGroups = append(costGroups, types.CostGroup{
			Name:   usageTypeName,
			Amount: amount,
		})
	}

	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	if len(costGroups) > 10 {
		costGroups = costGroups[:10]
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

	return types.CostData{Title: "📊 Top 10 Usage Types", Rows: rows}
}

// GetCurrentMonthData fetches current month cost breakdown
func GetCurrentMonthData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonthStart := currentMonthStart.AddDate(0, 1, 0)

	startDate := currentMonthStart.Format("2006-01-02")
	endDate := nextMonthStart.Format("2006-01-02")

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &awstypes.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"BlendedCost", "UnblendedCost", "NetUnblendedCost"},
	})

	rows := [][]string{
		{"Period", "Metric", "Amount"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return types.CostData{Title: "Current Month Breakdown", Rows: rows}
	}

	for _, resultByTime := range result.ResultsByTime {
		period := fmt.Sprintf("%s to %s", *resultByTime.TimePeriod.Start, *resultByTime.TimePeriod.End)

		if blendedCost, exists := resultByTime.Total["BlendedCost"]; exists {
			rows = append(rows, []string{period, "Total Blended Cost", formatCost(blendedCost.Amount)})
		}
		if unblendedCost, exists := resultByTime.Total["UnblendedCost"]; exists {
			rows = append(rows, []string{period, "Total Unblended Cost", formatCost(unblendedCost.Amount)})
		}
		if netCost, exists := resultByTime.Total["NetUnblendedCost"]; exists {
			rows = append(rows, []string{period, "Total Net Cost", formatCost(netCost.Amount)})
		}
	}

	return types.CostData{Title: fmt.Sprintf("📅 Current Month Costs (%s)", now.Format("January 2006")), Rows: rows}
}

// formatCost formats a cost amount string for display
func formatCost(amountStr *string) string {
	if amountStr == nil {
		return "$0.00"
	}

	amount, err := strconv.ParseFloat(*amountStr, 64)
	if err != nil {
		return "$0.00"
	}

	rounded := math.Ceil(amount*100) / 100
	return fmt.Sprintf("$%.2f", rounded)
}
