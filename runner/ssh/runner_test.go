package ssh

import (
	"bytes"
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/afs"
	"github.com/viant/scy/cred"
)

func TestService_Run(t *testing.T) {
	privateKeyBytes := getKeyLocation()
	if len(privateKeyBytes) == 0 {
		return
	}
	sshCred := cred.SSH{
		PrivateKeyPayload: privateKeyBytes,
		Basic: cred.Basic{
			Username: os.Getenv("USER"),
		},
	}
	clientConfig, err := sshCred.Config(context.Background())
	if !assert.Nil(t, err) {
		return
	}

	ctx := context.Background()
	runner := New("127.0.0.1:22", clientConfig)
	output, _, err := runner.Run(ctx, "ls /")
	assert.Nil(t, err)
	assert.Truef(t, len(output) > 0, "output was empty")
	assert.True(t, runner.PID() > 0)
}

func getKeyLocation() []byte {
	ctx := context.Background()
	fs := afs.New()
	knowHostLocation := path.Join(os.Getenv("HOME"), ".ssh/known_hosts")
	knownHosts, _ := fs.DownloadWithURL(ctx, knowHostLocation)
	if len(knownHosts) == 0 || !bytes.Contains(knownHosts, []byte("localhost")) {
		return nil
	}

	keyLocation := path.Join(os.Getenv("HOME"), ".ssh/id_rsa")
	exists, _ := fs.Exists(ctx, keyLocation)
	if !exists {
		return nil
	}
	keyBytes, _ := fs.DownloadWithURL(ctx, keyLocation)
	return keyBytes
}
