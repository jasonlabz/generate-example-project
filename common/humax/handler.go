package humax

import "context"

// Handler is a controller handler that returns domain response data.
type Handler[I, D any] func(context.Context, *I) (*D, error)

// Wrap adapts a controller handler to the shared success and error envelope.
func Wrap[I, D any](version string, handler Handler[I, D]) func(context.Context, *I) (*Output[D], error) {
	return func(ctx context.Context, input *I) (*Output[D], error) {
		data, err := handler(ctx, input)
		if err != nil {
			return nil, MapError(version, err)
		}
		var output D
		if data != nil {
			output = *data
		}
		return Success(version, output), nil
	}
}
