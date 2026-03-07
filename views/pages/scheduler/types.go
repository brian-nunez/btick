package scheduler

import "time"

type UIJob struct {
	ID             string
	Name           string
	Method         string
	URL            string
	HeadersJSON    string
	BodyJSON       string
	CronExpression string
	Timezone       string
	RetryMax       int32
	TimeoutSeconds int32
	Enabled        bool
	NextRunAt      *time.Time
	LastRunAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UIJobForm struct {
	Name           string
	Method         string
	URL            string
	HeadersJSON    string
	BodyJSON       string
	CronExpression string
	Timezone       string
	RetryMax       int32
	TimeoutSeconds int32
	Enabled        bool
}

type UIRun struct {
	ID              string
	JobID           string
	TriggerType     string
	AttemptNumber   int32
	Status          string
	RequestMethod   string
	RequestURL      string
	RequestHeaders  string
	RequestBody     string
	ResponseStatus  string
	ResponseHeaders string
	ResponseBody    string
	ErrorMessage    string
	StartedAt       time.Time
	FinishedAt      *time.Time
}

type UIAPIKey struct {
	ID         string
	Name       string
	KeyPrefix  string
	Scopes     string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
}

type JobsPageData struct {
	Title string
	Jobs  []UIJob
}

type JobsTableData struct {
	Jobs []UIJob
}

type JobFormPageData struct {
	Title       string
	SubmitLabel string
	FormAction  string
	Job         UIJobForm
	Error       string
}

type JobDetailPageData struct {
	Job  UIJob
	Runs []UIRun
}

type JobRunsPageData struct {
	Job  UIJob
	Runs []UIRun
}

type RunDetailPageData struct {
	Run UIRun
}

type APIKeysPageData struct {
	Keys []UIAPIKey
}

type CreateAPIKeyPageData struct {
	Error  string
	RawKey string
	Name   string
	Scopes string
}

type AuthPageData struct {
	Title            string
	Subtitle         string
	Action           string
	SubmitLabel      string
	SecondaryText    string
	SecondaryURL     string
	SecondaryLabel   string
	BackURL          string
	BackLabel        string
	Email            string
	EmailPlaceholder string
	Error            string
}
