package gosh_test

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/viant/afs"
	"github.com/viant/gosh"
	"github.com/viant/gosh/runner/local"
	"github.com/viant/gosh/runner/ssh"
	"github.com/viant/scy/cred"
	"os"
	"path"
	"testing"
)

func TestService_Run(t *testing.T) {
	srv, err := gosh.New(context.Background(), local.New())
	assert.Nil(t, err)
	assert.NotNil(t, srv.OsInfo())
	assert.NotNil(t, srv.HardwareInfo())

	_, _, err = srv.Run(context.Background(), "cd /etc")
	if err != nil {
		panic(err)
	}
	output, _, err := srv.Run(context.Background(), "ls -al")
	if err != nil {
		panic(err)
	}
	assert.True(t, len(output) > 0)
}

func ExampleLocalRun() {
	srv, err := gosh.New(context.Background(), local.New())
	if err != nil {
		return
	}
	_, _, err = srv.Run(context.Background(), "cd /etc")
	output, _, err := srv.Run(context.Background(), "ls -l")
	println(output)
}

func ExampleRemoveRun() {
	host := "localhost"
	privateKeyBytes := getKeyLocation(host)
	if privateKeyBytes == nil {
		return
	}
	sshCred := cred.SSH{
		PrivateKey: privateKeyBytes,
		Basic: cred.Basic{
			Username: os.Getenv("USER"),
		},
	}
	clientConfig, err := sshCred.Config(context.Background())
	if err != nil {
		panic(err)
	}
	srv, err := gosh.New(context.Background(), ssh.New(host+":22", clientConfig))
	if err != nil {
		return
	}
	_, _, err = srv.Run(context.Background(), "cd /etc")
	output, _, err := srv.Run(context.Background(), "ls -l")
	println(output)
}

func getKeyLocation(host string) []byte {
	ctx := context.Background()
	fs := afs.New()
	knowHostLocation := path.Join(os.Getenv("HOME"), ".ssh/known_hosts")
	knownHosts, _ := fs.DownloadWithURL(ctx, knowHostLocation)
	if len(knownHosts) == 0 || !bytes.Contains(knownHosts, []byte(host)) {
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
