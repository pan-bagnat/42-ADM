package ids

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropy     = ulid.Monotonic(rand.Reader, 0)
	entropyLock sync.Mutex
)

// New returns a prefixed ULID such as "adm_session_01H…".
func New(prefix string) (string, error) {
	entropyLock.Lock()
	defer entropyLock.Unlock()

	id, err := ulid.New(ulid.Timestamp(time.Now()), entropy)
	if err != nil {
		return "", fmt.Errorf("generate ulid: %w", err)
	}
	return fmt.Sprintf("%s_%s", prefix, id.String()), nil
}
