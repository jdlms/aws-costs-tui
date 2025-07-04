// forecast.go - forecast data fetching
package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func getForecastData(client *costexplorer.Client) CostData {
	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now()
	// Forecast for next month - exact logic from main.go
	nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	if nextMonthStart.Month() == 1 { // Handle year rollover
		nextMonthStart = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
	}
	nextMonthEnd := nextMonthStart.AddDate(0, 1, 0)

	startDate := nextMonthStart.Format("2006-01-02")
	endDate := nextMonthEnd.Format("2006-01-02")

	forecast, err := client.GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metric:      types.MetricBlendedCost,
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
		return CostData{Title: "Cost Forecast", Rows: rows}
	}

	period := fmt.Sprintf("%s to %s", startDate, endDate)
	rows = append(rows, []string{"Predicted Total Cost", formatCost(forecast.Total.Amount), period})

	if forecast.ForecastResultsByTime != nil {
		for _, forecastResult := range forecast.ForecastResultsByTime {
			forecastPeriod := fmt.Sprintf("%s to %s", *forecastResult.TimePeriod.Start, *forecastResult.TimePeriod.End)
			rows = append(rows, []string{"Mean Estimate", formatCost(forecastResult.MeanValue), forecastPeriod})
		}
	}

	return CostData{Title: "🔮 Cost Forecast", Rows: rows}
}
