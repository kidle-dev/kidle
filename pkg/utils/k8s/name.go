package k8s

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	// ResourceNameLengthLimit stores maximum allowed Length for a ResourceName
	ResourceNameLengthLimit = 63
)

// ToDNSName create a valid DNS name using a prefix, name and suffix. The name is based64 hashed if necessary
func ToDNSName(prefix string, name string, suffix string) string {
	if suffix != "" {
		suffix = fmt.Sprintf("-%s", suffix)
	}
	n := fmt.Sprintf("%s-%s%s", prefix, name, suffix)
	if len(n) <= 63 {
		return n
	}

	i := len(fmt.Sprintf("%s-%s", prefix, suffix))
	left := ResourceNameLengthLimit - i
	b64 := Base64(name, 6)
	return strings.ToLower(fmt.Sprintf("%s-%s-%s%s", prefix, name[:left-7], b64, suffix))
}

// Base64 returns the n first characters of a base64 encoding of src.
// It returns the full base64 if length equals 0
func Base64(src string, length int) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(src))
	if length >= 0 {
		return encoded[:length]
	}
	return encoded
}
