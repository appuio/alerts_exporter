package saauth_test

import (
	"os"
	"testing"
	"time"

	"github.com/appuio/alerts_exporter/internal/saauth"
	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ServiceAccountAuthInfoWriter_AuthenticateRequest(t *testing.T) {
	tokenFile := t.TempDir() + "/token"

	require.NoError(t, os.WriteFile(tokenFile, []byte("token"), 0644))

	subject, err := saauth.NewServiceAccountAuthInfoWriter(tokenFile, time.Millisecond)
	require.NoError(t, err)
	defer subject.Stop()

	r := new(runtime.TestClientRequest)
	require.NoError(t, subject.AuthenticateRequest(r, nil))
	require.Equal(t, "Bearer token", r.GetHeaderParams().Get("Authorization"))

	require.NoError(t, os.WriteFile(tokenFile, []byte("new-token"), 0644))
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		r := new(runtime.TestClientRequest)
		require.NoError(t, subject.AuthenticateRequest(r, nil))
		require.Equal(t, "Bearer new-token", r.GetHeaderParams().Get("Authorization"))
	}, 5*time.Second, time.Millisecond)
}

func Test_NewServiceAccountAuthInfoWriter_TokenReadErr(t *testing.T) {
	tokenFile := t.TempDir() + "/token"

	_, err := saauth.NewServiceAccountAuthInfoWriter(tokenFile, time.Millisecond)
	require.ErrorIs(t, err, os.ErrNotExist)
}
