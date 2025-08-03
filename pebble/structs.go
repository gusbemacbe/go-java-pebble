package pebble

// The `Profile` struct represents a user’s profile information
type Profile struct {
	URL string
}

// The `User` struct represents a user of the system
type User struct {
	Name string
	Age  int
	// Using a pointer to a struct to test nested access on pointers
	Profile *Profile
}
