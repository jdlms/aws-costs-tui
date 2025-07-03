# AWS Cost Explorer GO POC

This is a simple proof-of-concept application demonstrating various types of AWS billing data you can retrieve using the Cost Explorer API.

## Features Demonstrated

### 1. Current Month Costs

- **Blended Cost**: Your effective cost after applying volume discounts and Reserved Instance benefits
- **Unblended Cost**: The actual cost without any discounts applied
- **Net Unblended Cost**: Cost after credits and refunds
- All costs formatted as $X.XX with proper rounding

### 2. Forecasted Costs

- Predict next month's AWS spending using AWS's machine learning algorithms
- Can forecast up to 12 months in advance
- Provides mean estimates based on historical usage patterns
- Formatted as $X.XX for easy reading

### 3. Cost Breakdown by Dimensions

- **By Service**: EC2, S3, RDS, Lambda, etc.
- **By Region**: us-east-1, eu-west-1, etc.
- **By Usage Type**: Detailed usage categories like "EBS:VolumeUsage.gp2"
- All breakdowns sorted from highest to lowest cost
- Zero-cost items filtered out for cleaner output

### 4. Concurrent Execution

- All API calls run concurrently using goroutines for faster execution
- Significantly reduces total execution time compared to sequential calls
- Results are displayed as they arrive from the API
- Execution time is tracked and displayed

## Available Metrics

The Cost Explorer API supports several cost metrics:

- `BlendedCost`: Effective cost after applying RIs and volume discounts
- `UnblendedCost`: On-demand cost without discounts
- `NetUnblendedCost`: Cost after credits and refunds
- `UsageQuantity`: Amount of usage (hours, GB, requests, etc.)

## Available Dimensions for Grouping

You can group costs by various dimensions:

- `SERVICE`: AWS service names
- `AZ`: Availability Zone
- `INSTANCE_TYPE`: EC2 instance types
- `REGION`: AWS regions
- `USAGE_TYPE`: Detailed usage categories
- `USAGE_TYPE_GROUP`: Grouped usage types
- `RECORD_TYPE`: DiscountedUsage, Usage, Credit, etc.
- `OPERATING_SYSTEM`: Linux, Windows, etc.
- `TENANCY`: Shared, Dedicated, Host
- `SCOPE`: Regional or Global
- `PLATFORM`: EC2-Classic, EC2-VPC, etc.
- `SUBSCRIPTION_ID`: For Reserved Instances
- `LEGAL_ENTITY_NAME`: For consolidated billing
- `DEPLOYMENT_OPTION`: Single-AZ, Multi-AZ
- `DATABASE_ENGINE`: MySQL, PostgreSQL, etc.
- `CACHE_ENGINE`: Redis, Memcached
- `INSTANCE_TYPE_FAMILY`: m5, c5, r5, etc.

## Time Granularity Options

- `DAILY`: Daily cost breakdown
- `MONTHLY`: Monthly cost breakdown
- `HOURLY`: Hourly cost breakdown (limited to 7 days)

## Additional API Endpoints Available

1. **GetCostAndUsage**: Historical cost and usage data
2. **GetCostForecast**: Predicted future costs
3. **GetRightsizingRecommendation**: EC2 rightsizing suggestions
4. **GetReservationCoverage**: RI coverage analysis
5. **GetReservationPurchaseRecommendation**: RI purchase suggestions
6. **GetReservationUtilization**: RI utilization analysis
7. **GetUsageForecast**: Predicted future usage
8. **GetSavingsPlansUtilization**: Savings Plans utilization
9. **GetSavingsPlansUtilizationDetails**: Detailed Savings Plans data

## Prerequisites

1. AWS credentials configured (via `~/.aws/credentials` or environment variables)
2. Cost Explorer API access (may require enabling in AWS Console)
3. Appropriate IAM permissions for Cost Explorer API

## Required IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ce:GetCostAndUsage",
        "ce:GetCostForecast",
        "ce:GetUsageForecast",
        "ce:GetReservationCoverage",
        "ce:GetReservationPurchaseRecommendation",
        "ce:GetReservationUtilization",
        "ce:GetRightsizingRecommendation",
        "ce:GetSavingsPlansUtilization",
        "ce:GetSavingsPlansUtilizationDetails"
      ],
      "Resource": "*"
    }
  ]
}
```

## Running the Application

```bash
go mod tidy
go run main.go
```

## Sample Output

The application will show:

1. Current month total costs (blended, unblended, net) - formatted as $X.XX
2. Forecasted costs for next month - formatted as $X.XX
3. Cost breakdown by AWS region - sorted highest to lowest, formatted as $X.XX
4. Top 10 usage types by cost - sorted highest to lowest, formatted as $X.XX
5. Last 30 days costs broken down by service - sorted highest to lowest, formatted as $X.XX
6. Total execution time for all concurrent API calls

**Cost Formatting:**

- All costs are displayed in dollars and cents format ($4.45)
- Amounts are rounded up on the 3rd decimal place (e.g., 4.444 becomes $4.45)
- All cost groups are sorted from highest to lowest cost
- Zero-cost items are filtered out for cleaner output

**Performance:**

- All API calls execute concurrently using goroutines
- Typical execution time: 2-4 seconds (vs 8-12 seconds sequentially)
- Results display as they arrive from AWS APIs

## Cost Explorer Limitations

- **Data Latency**: Cost data is typically 24-48 hours behind
- **Forecast Accuracy**: Forecasts are estimates based on historical patterns
- **API Costs**: Cost Explorer API calls have their own charges (around $0.01 per request)
- **Rate Limits**: Standard AWS API rate limiting applies
