package domain

type Activity interface {
	GetData() ActivityData
	GetType() ActivityType
	GetSummary() string
}
