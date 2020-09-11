package options

type DockerOptions struct {
	NoPull       bool
	NoPullImages []string
}

func (do DockerOptions) ShouldPullImage(image string) bool {
	if do.NoPull {
		return false
	}

	for _, noPullImage := range do.NoPullImages {
		if noPullImage == image {
			return false
		}
	}

	return true
}
