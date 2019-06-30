package weather

// NetworkError networks related errors
type NetworkError string

func (e NetworkError) Error() string {
	return "NetworkError: " + string(e)
}

// ParseError parsing related errors
type ParseError string

func (e ParseError) Error() string {
	return "ParserError: " + string(e)
}
