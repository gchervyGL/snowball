package utils

import (
	"fmt"
	"time"
)

// HumanizeDuration convert duration to human string
func HumanizeDuration(duration time.Duration) string {
	if duration.Nanoseconds() < 1000 {
		return fmt.Sprintf("%d nanoseconds", int64(duration.Nanoseconds()))
	}
	if duration.Nanoseconds()/int64(time.Microsecond) < 1000 {
		return fmt.Sprintf("%d microseconds", int64(duration.Nanoseconds()/int64(time.Microsecond)))
	}
	if duration.Nanoseconds()/int64(time.Millisecond) < 1000 {
		return fmt.Sprintf("%d miliseconds", int64(duration.Nanoseconds()/int64(time.Millisecond)))
	}
	if duration.Nanoseconds() < 1000 {
		return fmt.Sprintf("%d nanoseconds", int64(duration.Nanoseconds()))
	}
	//if duration.Seconds() < 60.0 {
	//	return fmt.Sprintf("%d seconds", int64(duration.Seconds()))
	//}

	// only seconds
	return fmt.Sprintf("%d seconds", int64(duration.Seconds()))

	// if duration.Minutes() < 60.0 {
	// 	remainingSeconds := math.Mod(duration.Seconds(), 60)
	// 	return fmt.Sprintf("%d minutes %d seconds", int64(duration.Minutes()), int64(remainingSeconds))
	// }
	// if duration.Hours() < 24.0 {
	// 	remainingMinutes := math.Mod(duration.Minutes(), 60)
	// 	remainingSeconds := math.Mod(duration.Seconds(), 60)
	// 	return fmt.Sprintf("%d hours %d minutes %d seconds",
	// 		int64(duration.Hours()), int64(remainingMinutes), int64(remainingSeconds))
	// }
	// remainingHours := math.Mod(duration.Hours(), 24)
	// remainingMinutes := math.Mod(duration.Minutes(), 60)
	// remainingSeconds := math.Mod(duration.Seconds(), 60)
	// return fmt.Sprintf("%d days %d hours %d minutes %d seconds",
	// 	int64(duration.Hours()/24), int64(remainingHours),
	// 	int64(remainingMinutes), int64(remainingSeconds))
}
