package dockerfile

import (
	"context"
	"encoding/json"
	"github.com/moby/buildkit/client/llb/sourceresolver"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/frontend/dockerui"
	"github.com/moby/buildkit/solver/pb"
	"github.com/opencontainers/go-digest"
)

type DummyResolver struct{}

func (dr *DummyResolver) ResolveImageConfig(
	ctx context.Context,
	ref string,
	opt sourceresolver.Opt,
) (string, digest.Digest, []byte, error) {
	return ref, "", []byte("{}"), nil
}

func LocalContextSourcePaths(
	ctx context.Context,
	dockerfileContents []byte,
	dockerArguments map[string]string,
) ([]string, error) {
	var result []string

	state, _, _, _, err := dockerfile2llb.Dockerfile2LLB(context.Background(), dockerfileContents,
		dockerfile2llb.ConvertOpt{
			Config: dockerui.Config{
				BuildArgs: dockerArguments,
			},
			MetaResolver: &DummyResolver{},
		},
	)
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

		if source.Identifier != "local://"+dockerui.DefaultLocalNameContext {
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
