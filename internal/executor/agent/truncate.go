package agent

import "github.com/opencontainers/go-digest"

// maxImageHashCharacters contains a length of "IMAGE ID" field's in the `docker image ls` output.
//
// This length is large enough to not cause frequent collisions, but is more
// manageable by the users, compared to the full SHA-256 output in hex form.
const maxImageHashCharacters = 12

// truncate clamps a string to the provided length.
func truncate(s string, max int) string {
	actual := len(s)

	if max > actual {
		max = actual
	}

	return s[:max]
}

// truncateDigest parses OCI content digest and returns it's hash truncated
// to maxImageHashCharacters to make it more user-friendly.
func truncateDigest(digestString string) (string, error) {
	digestParsed, err := digest.Parse(digestString)
	if err != nil {
		return "", err
	}

	return truncate(digestParsed.Encoded(), maxImageHashCharacters), nil
}
