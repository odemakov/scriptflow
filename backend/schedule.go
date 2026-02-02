package main

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// durationMinMax returns -10% and +10% of given duration
func durationMinMax(duration time.Duration) (time.Duration, time.Duration) {
	spread := duration / 10
	return duration - spread, duration + spread
}

// cronFieldRanges defines the valid ranges for each cron field position
var cronFieldRanges = []struct {
	min, max int
}{
	{0, 59}, // minute
	{0, 23}, // hour
	{1, 31}, // day of month
	{1, 12}, // month
	{0, 6},  // day of week
}

// hashPattern matches H or H(min-max) in cron fields
var hashPattern = regexp.MustCompile(`^H(?:\((\d+)-(\d+)\))?$`)

// resolveHashedSchedule replaces Jenkins-style H notation with deterministic values
// H -> hash to full field range (e.g., 0-59 for minutes)
// H(10-30) -> hash to specified range
// Returns original schedule and error message if validation fails
func resolveHashedSchedule(schedule string, seed string) (string, error) {
	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return schedule, nil // not a standard 5-field cron, return as-is
	}

	h := fnv.New32a()
	h.Write([]byte(seed))
	hashValue := h.Sum32()

	resolved := make([]string, 5)
	fieldNames := []string{"minute", "hour", "day of month", "month", "day of week"}

	for i, field := range fields {
		matches := hashPattern.FindStringSubmatch(field)
		if matches == nil {
			resolved[i] = field
			continue
		}

		// Determine range
		minVal, maxVal := cronFieldRanges[i].min, cronFieldRanges[i].max
		if matches[1] != "" && matches[2] != "" {
			// H(min-max) syntax
			parsedMin, err := strconv.Atoi(matches[1])
			if err != nil {
				return schedule, fmt.Errorf("invalid min value in %s field: %s", fieldNames[i], matches[1])
			}
			parsedMax, err := strconv.Atoi(matches[2])
			if err != nil {
				return schedule, fmt.Errorf("invalid max value in %s field: %s", fieldNames[i], matches[2])
			}

			// Validate: min must be <= max (no wraparound)
			if parsedMin > parsedMax {
				return schedule, fmt.Errorf("invalid range in %s field: min (%d) > max (%d)", fieldNames[i], parsedMin, parsedMax)
			}

			// Validate: values must be within field bounds
			if parsedMin < cronFieldRanges[i].min || parsedMax > cronFieldRanges[i].max {
				return schedule, fmt.Errorf("range out of bounds in %s field: H(%d-%d), valid range is %d-%d",
					fieldNames[i], parsedMin, parsedMax, cronFieldRanges[i].min, cronFieldRanges[i].max)
			}

			minVal = parsedMin
			maxVal = parsedMax
		}

		// Calculate deterministic value within range
		rangeSize := maxVal - minVal + 1
		value := minVal + int(hashValue%uint32(rangeSize))
		resolved[i] = strconv.Itoa(value)

		// Use different hash bits for each field
		hashValue = hashValue*31 + uint32(i)
	}

	return strings.Join(resolved, " "), nil
}
