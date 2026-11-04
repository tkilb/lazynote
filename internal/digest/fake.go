package digest

import "context"

// Fake is a Digester for tests: it records each source it is given and
// returns Result or Err. It never calls a model or the network.
type Fake struct {
	Result Digest
	Err    error
	// Calls holds the source text of each call, in order.
	Calls []string
}

// Digest implements Digester. A cancelled context returns its error first.
func (f *Fake) Digest(ctx context.Context, source string) (Digest, error) {
	if err := ctx.Err(); err != nil {
		return Digest{}, err
	}
	f.Calls = append(f.Calls, source)
	if f.Err != nil {
		return Digest{}, f.Err
	}
	return f.Result, nil
}
