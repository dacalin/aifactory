package sqlite

type scanner interface {
	Scan(...any) error
}
