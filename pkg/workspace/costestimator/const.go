package costestimator

import "time"

const (
	TimeInterval = 24 // Hours (One day)
)

func GetNumberOfDays() int {
	currentTime := time.Now()

	firstDayOfNextMonth := time.Date(
		currentTime.Year(),
		currentTime.Month()+1,
		1,
		0, 0, 0, 0,
		currentTime.Location(),
	)

	lastDayOfCurrentMonth := firstDayOfNextMonth.Add(-24 * time.Hour)

	numDaysInCurrentMonth := lastDayOfCurrentMonth.Day()
	return numDaysInCurrentMonth
}
