package saauth

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// NewServiceAccountAuthInfoWriter creates a new ServiceAccountAuthInfoWriter.
// ServiceAccountAuthInfoWriter implements Kubernetes service account authentication.
// It reads the token from the given file and refreshes it every refreshInterval.
// If refreshInterval is 0, it defaults to 5 minutes.
// If saFile is empty, it defaults to /var/run/secrets/kubernetes.io/serviceaccount/token.
// An error is returned if the initial token read fails. Further read failures do not cause an error.
func NewServiceAccountAuthInfoWriter(saFile string, refreshInterval time.Duration) (*ServiceAccountAuthInfoWriter, error) {
	if saFile == "" {
		saFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	}
	if refreshInterval == 0 {
		refreshInterval = 5 * time.Minute
	}

	w := &ServiceAccountAuthInfoWriter{
		ticker: time.NewTicker(refreshInterval),
		saFile: saFile,
	}

	t, err := w.readTokenFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read token from file: %w", err)
	}
	w.storeToken(t)

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.ticker.C:
				t, err := w.readTokenFromFile()
				if err != nil {
					log.Printf("failed to read token from file: %v", err)
					continue
				}
				w.storeToken(t)
			}
		}
	}()

	return w, nil
}

// ServiceAccountAuthInfoWriter implements Kubernetes service account authentication.
type ServiceAccountAuthInfoWriter struct {
	saFile string
	token  atomic.Value
	ticker *time.Ticker
	cancel context.CancelFunc
}

// AuthenticateRequest implements the runtime.ClientAuthInfoWriter interface.
// It sets the Authorization header to the current token.
func (s *ServiceAccountAuthInfoWriter) AuthenticateRequest(r runtime.ClientRequest, _ strfmt.Registry) error {
	return r.SetHeaderParam(runtime.HeaderAuthorization, "Bearer "+s.loadToken())
}

// Stop stops the token refresh
func (s *ServiceAccountAuthInfoWriter) Stop() {
	s.cancel()
	s.ticker.Stop()
}

func (s *ServiceAccountAuthInfoWriter) storeToken(t string) {
	s.token.Store(t)
}

func (s *ServiceAccountAuthInfoWriter) loadToken() string {
	return s.token.Load().(string)
}

func (s *ServiceAccountAuthInfoWriter) readTokenFromFile() (string, error) {
	t, err := os.ReadFile(s.saFile)
	return string(t), err
}
