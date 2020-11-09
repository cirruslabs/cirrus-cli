package options

type ContainerOptions struct {
	NoPull       bool
	NoPullImages []string
	NoCleanup    bool
}

func (do ContainerOptions) ShouldPullImage(image string) bool {
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
