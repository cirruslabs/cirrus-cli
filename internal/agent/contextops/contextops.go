package contextops

import "context"

func And(inputCtxs ...context.Context) context.Context {
	outputCtx, outputCtxCancel := context.WithCancel(context.Background())

	go func() {
		defer outputCtxCancel()

		for _, inputCtx := range inputCtxs {
			<-inputCtx.Done()
		}
	}()

	return outputCtx
}
