package picker

import (
	"errors"
	"fmt"

	"github.com/ktr0731/go-fuzzyfinder"
)

var ErrAborted = errors.New("no selection")

// Pick shows an fzf-like finder over items, which must already be ranked best-first:
// the first item is drawn next to the prompt and holds the cursor.
func Pick[T any](items []T, prompt, header string, label func(T) string) (T, error) {
	var zero T
	if len(items) == 0 {
		return zero, errors.New("nothing to pick from")
	}

	idx, err := fuzzyfinder.Find(items, func(i int) string { return label(items[i]) },
		fuzzyfinder.WithPromptString(prompt+"> "),
		fuzzyfinder.WithHeader(header),
	)
	if errors.Is(err, fuzzyfinder.ErrAbort) {
		return zero, ErrAborted
	}
	if err != nil {
		return zero, fmt.Errorf("finder: %w", err)
	}
	return items[idx], nil
}
