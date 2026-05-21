package application

type ReportEventType string

const (
	ReportEventTypeUserReport ReportEventType = "USER_REPORT"
	ReportEventTypeComplete   ReportEventType = "COMPLETE"
	ReportEventTypeError      ReportEventType = "ERROR"
)

type ReportEvent interface {
	GetType() ReportEventType
}

type UserReportEvent struct {
	Report UserReport
}

func (e *UserReportEvent) GetType() ReportEventType { return ReportEventTypeUserReport }

type ReportCompleteEvent struct{}

func (e *ReportCompleteEvent) GetType() ReportEventType { return ReportEventTypeComplete }

type ReportErrorEvent struct {
	Message string
}

func (e *ReportErrorEvent) GetType() ReportEventType { return ReportEventTypeError }
