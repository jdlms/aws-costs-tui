package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
)

// NewClient creates a new AWS Cost Explorer client
func NewClient() (*costexplorer.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return costexplorer.NewFromConfig(cfg), nil
}
