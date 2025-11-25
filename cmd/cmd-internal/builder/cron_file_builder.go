package builder

import (
	"fmt"
	"strings"
	"time"

	"github.com/martijn/dbcalm-cmd/cmd-internal/model"
)

type CronFileBuilder struct {
	projectName string
}

func NewCronFileBuilder(projectName string) *CronFileBuilder {
	return &CronFileBuilder{
		projectName: projectName,
	}
}

// GenerateCronExpression generates a cron expression from a schedule.
//
// Cron format: minute hour day_of_month month day_of_week
// For intervals: */X for minutes, or */X in hour field with * in minute field for hours
// For hourly: minute * * * * (run at specified minute every hour)
func (c *CronFileBuilder) GenerateCronExpression(schedule *model.Schedule) (string, error) {
	if schedule.Frequency == "interval" {
		if schedule.IntervalUnit == nil {
			return "", fmt.Errorf("interval_unit is required for interval frequency")
		}
		if schedule.IntervalValue == nil {
			return "", fmt.Errorf("interval_value is required for interval frequency")
		}

		if *schedule.IntervalUnit == "minutes" {
			// Run every X minutes: */X * * * *
			return fmt.Sprintf("*/%d * * * *", *schedule.IntervalValue), nil
		}
		if *schedule.IntervalUnit == "hours" {
			// Run every X hours: 0 */X * * *
			return fmt.Sprintf("0 */%d * * *", *schedule.IntervalValue), nil
		}
		return "", fmt.Errorf("unknown interval unit: %s", *schedule.IntervalUnit)
	}

	if schedule.Minute == nil {
		return "", fmt.Errorf("minute is required for non-interval frequencies")
	}
	minute := fmt.Sprintf("%d", *schedule.Minute)

	if schedule.Frequency == "hourly" {
		// Run once every hour at the specified minute: minute * * * *
		return fmt.Sprintf("%s * * * *", minute), nil
	}

	if schedule.Hour == nil {
		return "", fmt.Errorf("hour is required for daily/weekly/monthly frequencies")
	}
	hour := fmt.Sprintf("%d", *schedule.Hour)

	switch schedule.Frequency {
	case "daily":
		return fmt.Sprintf("%s %s * * *", minute, hour), nil
	case "weekly":
		dayOfWeek := "*"
		if schedule.DayOfWeek != nil {
			dayOfWeek = fmt.Sprintf("%d", *schedule.DayOfWeek)
		}
		return fmt.Sprintf("%s %s * * %s", minute, hour, dayOfWeek), nil
	case "monthly":
		dayOfMonth := "*"
		if schedule.DayOfMonth != nil {
			dayOfMonth = fmt.Sprintf("%d", *schedule.DayOfMonth)
		}
		return fmt.Sprintf("%s %s %s * *", minute, hour, dayOfMonth), nil
	default:
		return "", fmt.Errorf("unknown frequency: %s", schedule.Frequency)
	}
}

// GenerateCronCommand generates the dbcalm backup command that will be executed by cron.
func (c *CronFileBuilder) GenerateCronCommand(schedule *model.Schedule) string {
	// Build command to call dbcalm backup CLI with schedule_id
	backupCmd := fmt.Sprintf("/usr/bin/dbcalm backup %s", schedule.BackupType)
	scheduleArg := fmt.Sprintf("--schedule-id %d", schedule.ID)
	logFile := fmt.Sprintf("/var/log/%s/cron-%d.log", c.projectName, schedule.ID)
	logRedirect := fmt.Sprintf(">> %s 2>&1", logFile)
	return fmt.Sprintf("%s %s %s", backupCmd, scheduleArg, logRedirect)
}

// BuildCronFileContent builds complete cron file content from list of schedules.
//
// Only includes enabled schedules.
// Returns complete file content as string.
func (c *CronFileBuilder) BuildCronFileContent(schedules []model.Schedule) string {
	// Filter to only enabled schedules
	var enabledSchedules []model.Schedule
	for _, s := range schedules {
		if s.Enabled {
			enabledSchedules = append(enabledSchedules, s)
		}
	}

	// Build header
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	lines := []string{
		"# DBCalm Backup Schedules",
		"# Auto-generated - do not edit manually",
		fmt.Sprintf("# Last updated: %s", timestamp),
		"",
	}

	// Add daily cleanup job (runs at 2:00 AM)
	lines = append(lines, "# Daily cleanup job")
	lines = append(lines, fmt.Sprintf(
		"0 2 * * * root /usr/bin/dbcalm cleanup >> /var/log/%s/cleanup.log 2>&1",
		c.projectName,
	))
	lines = append(lines, "")

	// Add each schedule
	for _, schedule := range enabledSchedules {
		cronExpression, err := c.GenerateCronExpression(&schedule)
		if err != nil {
			// Log error but continue with other schedules
			lines = append(lines, fmt.Sprintf("# ERROR for schedule %d: %s", schedule.ID, err.Error()))
			lines = append(lines, "")
			continue
		}
		cronCommand := c.GenerateCronCommand(&schedule)

		lines = append(lines, fmt.Sprintf("# Schedule ID: %d", schedule.ID))
		lines = append(lines, fmt.Sprintf("%s root %s", cronExpression, cronCommand))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
