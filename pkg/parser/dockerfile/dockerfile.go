package dockerfile

import (
	"context"
	"encoding/json"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/solver/pb"
	"github.com/opencontainers/go-digest"
)

type DummyResolver struct{}

func (dr *DummyResolver) ResolveImageConfig(
	ctx context.Context,
	ref string,
	opt llb.ResolveImageConfigOpt,
) (digest.Digest, []byte, error) {
	return "", []byte("{}"), nil
}

func LocalContextSourcePaths(
	ctx context.Context,
	dockerfileContents []byte,
	dockerArguments map[string]string,
) ([]string, error) {
	var result []string

	const localContextName = "context"

	state, _, _, err := dockerfile2llb.Dockerfile2LLB(context.Background(), dockerfileContents, dockerfile2llb.ConvertOpt{
		MetaResolver:     &DummyResolver{},
		BuildArgs:        dockerArguments,
		ContextLocalName: localContextName,
	})
	if err != nil {
		return nil, err
	}

	marshalledState, err := state.Marshal(ctx)
	if err != nil {
		return nil, err
	}

	for _, dt := range marshalledState.Def {
		var op pb.Op

		if err := (&op).Unmarshal(dt); err != nil {
			return nil, err
		}

		source := op.GetSource()
		if source == nil {
			continue
		}

		if source.Identifier != "local://"+localContextName {
			continue
		}

		followPathsJSON, ok := source.Attrs[pb.AttrFollowPaths]
		if !ok {
			continue
		}

		var followPaths []string
		if err := json.Unmarshal([]byte(followPathsJSON), &followPaths); err != nil {
			return nil, err
		}

		result = append(result, followPaths...)
	}

	return result, nil
}
