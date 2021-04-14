package pointer

// Int32 returns a pointer to an Int32
func Int32(i int32) *int32 {
	return &i
}

// Int64 returns a pointer to an Int64
func Int64(i int64) *int64 {
	return &i
}

// Bool returns a pointer to a bool
func Bool(b bool) *bool {
	return &b
}
