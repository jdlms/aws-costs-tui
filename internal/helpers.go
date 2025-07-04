package internal

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// Helper functions from your original code
func FormatCost(amountStr *string) string {
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

func ParseAmountString(amountStr *string) float64 {
	if amountStr == nil {
		return 0.0
	}
	amount, err := strconv.ParseFloat(*amountStr, 64)
	if err != nil {
		return 0.0
	}
	return amount
}

func SortCostGroups(groups []types.Group) []CostGroup {
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

	sort.Slice(costGroups, func(i, j int) bool {
		return costGroups[i].Amount > costGroups[j].Amount
	})

	return costGroups
}
