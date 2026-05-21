package domain

type TimePeriod string

const (
	TimePeriodLast7Days       TimePeriod = "LAST_7_DAYS"
	TimePeriodLast30Days      TimePeriod = "LAST_30_DAYS"
	TimePeriodCurrentQuarter  TimePeriod = "CURRENT_QUARTER"
	TimePeriodLastQuarter     TimePeriod = "LAST_QUARTER"
	TimePeriodCurrentYear     TimePeriod = "CURRENT_YEAR"
	TimePeriodLastYear        TimePeriod = "LAST_YEAR"
	TimePeriodCustom          TimePeriod = "CUSTOM"
)
