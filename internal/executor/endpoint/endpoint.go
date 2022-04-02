package endpoint

type Endpoint interface {
	Container() string
	Direct() string
}

type Local struct {
	containerEndpoint string
	directEndpoint    string
}

func NewLocal(containerEndpoint string, directEndpoint string) *Local {
	return &Local{
		containerEndpoint: containerEndpoint,
		directEndpoint:    directEndpoint,
	}
}

func (local *Local) Container() string {
	return local.containerEndpoint
}

func (local *Local) Direct() string {
	return local.directEndpoint
}

type Remote struct {
	remoteEndpoint string
}

func NewRemote(remoteEndpoint string) *Remote {
	return &Remote{
		remoteEndpoint: remoteEndpoint,
	}
}

func (remote *Remote) Container() string {
	return remote.remoteEndpoint
}

func (remote *Remote) Direct() string {
	return remote.remoteEndpoint
}
