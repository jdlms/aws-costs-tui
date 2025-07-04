// dashboard.go - dashboard data fetching
package aws

import (
	"context"
	"fmt"
	"time"

	"cost-explorer/internal"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// GetDashboardData fetches dashboard cost data - exact patterns from main.go
func GetDashboardData(client *costexplorer.Client) internal.CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	// Get first day of current month
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// Get first day of next month
	nextMonthStart := currentMonthStart.AddDate(0, 1, 0)

	startDate := currentMonthStart.Format("2006-01-02")
	endDate := nextMonthStart.Format("2006-01-02")

	result, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
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
		return internal.CostData{Title: "Dashboard Overview", Rows: rows}
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

	return internal.CostData{Title: "💸 Dashboard Overview", Rows: rows}
}
