package apiresp

type ValidationError struct {
	Parameter string       `json:"parameter"`
	Location  LocationType `json:"location"`
	Expected  string       `json:"expected"`
	Received  string       `json:"received"`
	Message   string       `json:"message"`
}

// NewValidationError encapsulates instantiation of a new ValidationError non-pointer value
//
//nolint:staticcheck // S1016: Keep Args/Error types separate to allow independent evolution
func NewValidationError(args ValidationErrorArgs) ValidationError {
	return ValidationError{
		Parameter: args.Parameter,
		Location:  args.Location,
		Expected:  args.Expected,
		Received:  args.Received,
		Message:   args.Message,
	}
}

type ValidationErrorArgs struct {
	Parameter string
	Location  LocationType
	Expected  string
	Received  string
	Message   string
}
