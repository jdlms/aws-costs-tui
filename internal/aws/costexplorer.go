package aws

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"cost-explorer/internal/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	awstypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// getCurrentMonthPeriod returns the now month date interval
func getCurrentMonthPeriod() awstypes.DateInterval {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0) // AWS expects exclusive end date

	return awstypes.DateInterval{
		Start: aws.String(start.Format("2006-01-02")),
		End:   aws.String(end.Format("2006-01-02")),
	}
}

// getPreviousMonthPeriod returns the previous month date interval
func getPreviousMonthPeriod() awstypes.DateInterval {
	now := time.Now()
	start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return awstypes.DateInterval{
		Start: aws.String(start.Format("2006-01-02")),
		End:   aws.String(end.Format("2006-01-02")),
	}
}

// getNextMonthPeriod returns the next month date interval
func getNextMonthPeriod() awstypes.DateInterval {
	now := time.Now()
	start := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return awstypes.DateInterval{
		Start: aws.String(start.Format("2006-01-02")),
		End:   aws.String(end.Format("2006-01-02")),
	}
}

// GetDashboardData fetches dashboard overview data with now month and forecast
func GetDashboardData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows := [][]string{
		{"Period", "Cost Type", "Amount"},
	}

	// Get now month data
	currentPeriod := getCurrentMonthPeriod()
	currentResult, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod:  &currentPeriod,
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"NetUnblendedCost"},
	})

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Current Month", "Timeout", "Request timed out after 30 seconds"})
		} else {
			rows = append(rows, []string{"Current Month", "Error", err.Error()})
		}
	} else {
		for _, resultByTime := range currentResult.ResultsByTime {
			currentMonthName := time.Now().Format("January 2006")
			if netCost, exists := resultByTime.Total["NetUnblendedCost"]; exists {
				rows = append(rows, []string{currentMonthName, "Current Month Total", formatCost(netCost.Amount)})
			}
		}
	}

	// Get forecast data for now month (month-to-date projection)
	now := time.Now()
	currentMonthEnd := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	forecastPeriod := awstypes.DateInterval{
		Start: aws.String(now.Format("2006-01-02")),             // From today
		End:   aws.String(currentMonthEnd.Format("2006-01-02")), // To end of now month
	}

	forecast, err := client.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
		TimePeriod:  &forecastPeriod,
		Granularity: awstypes.GranularityMonthly,
		Metric:      awstypes.MetricNetUnblendedCost,
	})

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Current Month", "Timeout", "Request timed out after 30 seconds"})
		} else {
			rows = append(rows, []string{"Current Month", "Error", err.Error()})
		}
	} else {
		currentMonthName := time.Now().Format("January 2006")
		rows = append(rows, []string{currentMonthName, "Forecasted Total", formatCost(forecast.Total.Amount)})
	}

	return types.CostData{Title: "💸 Dashboard Overview", Rows: rows}
}

// GetForecastData fetches cost forecast data
func GetForecastData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	period := getNextMonthPeriod()

	forecast, err := client.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
		TimePeriod:  &period,
		Granularity: awstypes.GranularityMonthly,
		Metric:      awstypes.MetricNetUnblendedCost,
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

	periodStr := fmt.Sprintf("%s to %s", *period.Start, *period.End)
	rows = append(rows, []string{"Predicted Total Cost", formatCost(forecast.Total.Amount), periodStr})

	if forecast.ForecastResultsByTime != nil {
		for _, forecastResult := range forecast.ForecastResultsByTime {
			forecastPeriod := fmt.Sprintf("%s to %s", *forecastResult.TimePeriod.Start, *forecastResult.TimePeriod.End)
			rows = append(rows, []string{"Mean Estimate", formatCost(forecastResult.MeanValue), forecastPeriod})
		}
	}

	return types.CostData{Title: "🔮 Cost Forecast", Rows: rows}
}

// getThreeMonthPeriod returns a date interval covering now month and previous two months
func getThreeMonthPeriod() awstypes.DateInterval {
	now := time.Now()
	// Start from 2 months ago
	start := time.Date(now.Year(), now.Month()-2, 1, 0, 0, 0, 0, time.UTC)
	// End at the start of next month (AWS expects exclusive end date)
	end := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	return awstypes.DateInterval{
		Start: aws.String(start.Format("2006-01-02")),
		End:   aws.String(end.Format("2006-01-02")),
	}
}

// normalizeServiceName standardizes service names to match console display
func normalizeServiceName(serviceName string) string {
	// Common service name mappings from API to console display names
	serviceMap := map[string]string{
		"Amazon Elastic Compute Cloud - Compute": "Amazon Elastic Compute Cloud",
		"Amazon Elastic Container Service":       "Amazon ECS",
		"Amazon EC2 Container Service":           "Amazon ECS",
		"Amazon Elastic Load Balancing":          "Elastic Load Balancing",
		"AWS Data Transfer":                      "Data Transfer",
		"Amazon CloudFront":                      "CloudFront",
		"Amazon Virtual Private Cloud":           "Amazon VPC",
		"AWS WAF":                                "AWS WAF",
		"Amazon Relational Database Service":     "Amazon RDS",
		// Additional potential Data Transfer variations
		"Data Transfer":        "Data Transfer",
		"AWS DataTransfer":     "Data Transfer",
		"Amazon Data Transfer": "Data Transfer",
		"EC2 - Other":          "Data Transfer", // Data transfer costs appear here
		"EC2-Other":            "Data Transfer", // Alternative format
		"Amazon Elastic Compute Cloud - Data Transfer": "Data Transfer",
	}

	if normalized, exists := serviceMap[serviceName]; exists {
		return normalized
	}
	return serviceName
}

// isTaxService checks if a service name represents tax and should be excluded
func isTaxService(serviceName string) bool {
	taxServices := []string{
		"Tax",
		"AWS Tax",
		"Amazon Tax",
		"Sales Tax",
		"VAT",
		"GST",
		"Tax Service",
		"Taxation",
		"AWS Sales Tax",
		"Amazon Sales Tax",
	}

	for _, taxService := range taxServices {
		if serviceName == taxService {
			return true
		}
	}
	return false
}

// GetServiceData fetches costs grouped by service for now month and previous two months
func GetServiceData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	period := getThreeMonthPeriod()
	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod:  &period,
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"NetUnblendedCost"},
		GroupBy: []awstypes.GroupDefinition{{
			Type: awstypes.GroupDefinitionTypeDimension,
			Key:  &[]string{"SERVICE"}[0],
		}},
	})

	now := time.Now()
	currentMonthKey := now.Format("Jan")
	prevMonth1 := now.AddDate(0, -1, 0).Format("Jan")
	prevMonth2 := now.AddDate(0, -2, 0).Format("Jan")

	rows := [][]string{
		{"Service", "Now", prevMonth1, prevMonth2},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Current Month", "Timeout", "Request timed out after 30 seconds"})
		} else {
			rows = append(rows, []string{"Current Month", "Error", err.Error()})
		}
		return types.CostData{Title: "Costs by Service", Rows: rows}
	}

	// Map to store service costs by month: service -> month -> cost
	serviceMonthCosts := make(map[string]map[string]float64)
	monthTotals := make(map[string]float64)
	for _, resultByTime := range result.ResultsByTime {
		// Parse the month from the time period
		startDate, err := time.Parse("2006-01-02", *resultByTime.TimePeriod.Start)
		if err != nil {
			continue
		}
		monthKey := startDate.Format("Jan")

		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && group.Metrics != nil {
				rawServiceName := group.Keys[0]
				serviceName := normalizeServiceName(rawServiceName)

				// Skip tax services
				if isTaxService(serviceName) || isTaxService(rawServiceName) {
					continue
				}

				if netCost, exists := group.Metrics["NetUnblendedCost"]; exists && netCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*netCost.Amount, 64); err == nil && amount > 0 {
						if serviceMonthCosts[serviceName] == nil {
							serviceMonthCosts[serviceName] = make(map[string]float64)
						}
						serviceMonthCosts[serviceName][monthKey] += amount // Add to existing amount instead of overwriting
						monthTotals[monthKey] += amount
					}
				}
			}
		}
	}

	// Create service groups for sorting
	type ServiceData struct {
		Name   string
		Month0 float64 // Current month
		Month1 float64 // Previous month
		Month2 float64 // 2 months ago
	}

	var services []ServiceData
	for serviceName, monthCosts := range serviceMonthCosts {
		service := ServiceData{
			Name:   serviceName,
			Month0: monthCosts[currentMonthKey],
			Month1: monthCosts[prevMonth1],
			Month2: monthCosts[prevMonth2],
		}
		services = append(services, service)
	}

	// Sort by now month cost first (primary), then by previous months
	// Services with now month costs come first, then services with only past costs
	sort.Slice(services, func(i, j int) bool {
		// If one has now month cost and the other doesn't, prioritize now month
		if services[i].Month0 > 0 && services[j].Month0 == 0 {
			return true
		}
		if services[i].Month0 == 0 && services[j].Month0 > 0 {
			return false
		}

		// If both have now month costs, sort by now month cost descending
		if services[i].Month0 > 0 && services[j].Month0 > 0 {
			return services[i].Month0 > services[j].Month0
		}

		// If neither has now month costs, sort by most recent past month cost
		if services[i].Month1 != services[j].Month1 {
			return services[i].Month1 > services[j].Month1
		}
		return services[i].Month2 > services[j].Month2
	})

	// Add service rows
	for _, service := range services {
		month0Str := "$0.00" // Now month
		month1Str := "$0.00" // Previous month
		month2Str := "$0.00" // 2 months ago

		if service.Month0 > 0 {
			amount := strconv.FormatFloat(service.Month0, 'f', -1, 64)
			month0Str = formatCost(&amount)
		}
		if service.Month1 > 0 {
			amount := strconv.FormatFloat(service.Month1, 'f', -1, 64)
			month1Str = formatCost(&amount)
		}
		if service.Month2 > 0 {
			amount := strconv.FormatFloat(service.Month2, 'f', -1, 64)
			month2Str = formatCost(&amount)
		}

		rows = append(rows, []string{
			service.Name,
			month0Str, // Now month first
			month1Str, // Previous month second
			month2Str, // 2 months ago third
		})
	}

	return types.CostData{Title: "🛠️Services", Rows: rows}
}

// GetRegionData fetches costs grouped by region
func GetRegionData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	period := getCurrentMonthPeriod()

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod:  &period,
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"NetUnblendedCost"},
		GroupBy: []awstypes.GroupDefinition{
			{
				Type: awstypes.GroupDefinitionTypeDimension,
				Key:  &[]string{"REGION"}[0],
			},
		},
	})

	rows := [][]string{
		{"Region", "Cost (Current Month)", "Percentage"},
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rows = append(rows, []string{"Timeout", "Request timed out after 30 seconds", ""})
		} else {
			rows = append(rows, []string{"Error", err.Error(), ""})
		}
		return types.CostData{Title: "Regions", Rows: rows}
	}

	var totalCost float64
	regionMap := make(map[string]float64)

	for _, resultByTime := range result.ResultsByTime {
		for _, group := range resultByTime.Groups {
			if len(group.Keys) > 0 && group.Metrics != nil {
				regionName := group.Keys[0]
				if netCost, exists := group.Metrics["NetUnblendedCost"]; exists && netCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*netCost.Amount, 64); err == nil && amount > 0 {
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

	return types.CostData{Title: "🌍 Regions", Rows: rows}
}

// GetUsageTypeData fetches costs grouped by usage type
func GetUsageTypeData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	period := getCurrentMonthPeriod()

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod:  &period,
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"NetUnblendedCost"},
		GroupBy: []awstypes.GroupDefinition{
			{
				Type: awstypes.GroupDefinitionTypeDimension,
				Key:  &[]string{"USAGE_TYPE"}[0],
			},
		},
	})

	rows := [][]string{
		{"Usage Type", "Cost (Current Month)", "Percentage"},
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
				if netCost, exists := group.Metrics["NetUnblendedCost"]; exists && netCost.Amount != nil {
					if amount, err := strconv.ParseFloat(*netCost.Amount, 64); err == nil && amount > 0 {
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

// GetCurrentMonthData fetches now month cost breakdown
func GetCurrentMonthData(client *costexplorer.Client) types.CostData {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	period := getCurrentMonthPeriod()

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod:  &period,
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

	return types.CostData{Title: fmt.Sprintf("📅 Current Month Costs (%s)", time.Now().Format("January 2006")), Rows: rows}
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

	// Use standard rounding instead of always rounding up
	return fmt.Sprintf("$%.2f", amount)
}
