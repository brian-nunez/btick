package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func NextRun(cronExpression string, timezone string, from time.Time) (time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load timezone: %w", err)
	}

	schedule, err := parser.Parse(cronExpression)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cron expression: %w", err)
	}

	return schedule.Next(from.In(location)).UTC(), nil
}

func ValidateCron(cronExpression string) error {
	_, err := parser.Parse(cronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}

func ValidateTimezone(timezone string) error {
	_, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}
	return nil
}
