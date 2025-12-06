package localsvr

type OnFailure string

const (
	ErrorOnFailure  OnFailure = "error"
	WarnOnFailure   OnFailure = "warn"
	IgnoreOnFailure OnFailure = "ignore"
)
