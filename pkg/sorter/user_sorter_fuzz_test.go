package sorter

import (
	"testing"
)

func FuzzParseUserSortField(f *testing.F) {
	f.Add("lastName")
	f.Add("last_name")
	f.Add("USERID")
	f.Add("email")
	f.Add("firstName")
	f.Add("")
	f.Add("unknown_field")

	f.Fuzz(func(t *testing.T, input string) {
		_ = parseUserSortField(input)
	})
}

func FuzzParseSortDirection(f *testing.F) {
	f.Add("ASC")
	f.Add("DESC")
	f.Add("asc")
	f.Add("ASCENDING")
	f.Add("DESCENDING")
	f.Add("")
	f.Add("invalid")

	f.Fuzz(func(t *testing.T, input string) {
		_ = parseSortDirection(input)
	})
}
