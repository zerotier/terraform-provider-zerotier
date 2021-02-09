package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

var (
	waitTime     = 2 * time.Second
	errInvalidIP = errors.New("invalid IP address")
)

func waitForIP(ctx context.Context, docker *client.Client, containerID string) (net.IP, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ip, err := getIP(ctx, docker, containerID)
		if err == errInvalidIP {
			time.Sleep(waitTime)
		} else if err == nil {
			return ip, nil
		} else {
			return nil, err
		}
	}
}

func getIP(ctx context.Context, docker *client.Client, containerID string) (net.IP, error) {
	res, err := execContainer(ctx, docker, containerID, "zerotier-cli", "listnetworks")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(res)), "\n")
	parts := strings.Split(lines[len(lines)-1], " ")
	if len(lines) < 2 {
		return nil, errInvalidIP // force a retry, we haven't joined the network yet
	}
	if len(parts) != 9 {
		return nil, fmt.Errorf("invalid data from listnetworks: line was %d parts: %q", len(parts), strings.Join(parts, " "))
	}

	ips := strings.Split(parts[8], ",")
	ip, _, err := net.ParseCIDR(ips[len(ips)-1])
	if ip == nil || err != nil {
		return nil, errInvalidIP
	}

	return ip, nil
}

func execContainer(ctx context.Context, docker *client.Client, containerID string, command ...string) ([]byte, error) {
	exec, err := docker.ContainerExecCreate(
		ctx,
		containerID,
		types.ExecConfig{
			Cmd:          command,
			AttachStdout: true,
			AttachStderr: true,
		},
	)
	if err != nil {
		return nil, err
	}

	r, err := docker.ContainerExecAttach(ctx, exec.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	errChan := make(chan error, 1)

	go func() {
		if _, err := stdcopy.StdCopy(buf, buf, r.Reader); err != nil {
			errChan <- err
			return
		}

		errChan <- nil
	}()

	if err := docker.ContainerExecStart(ctx, exec.ID, types.ExecStartCheck{}); err != nil {
		return nil, err
	}

	err = <-errChan

	res, err := docker.ContainerExecInspect(ctx, exec.ID)
	if err != nil {
		return nil, err
	}

	if res.ExitCode != 0 {
		return nil, errors.New("invalid exit code")
	}

	return buf.Bytes(), err
}

func TestDockerPing(t *testing.T) {
	tf := getTFTest(t)
	tf.Apply("testdata/plans/docker-integration.tf")

	var aliceCID, bobCID string

	// this extracts the container ids from the state so we can mess with them.
	resources := a(tf.State()["resources"])
	for _, resource := range resources {
		m := h(resource)
		if s(m["type"]) == "docker_container" {
			name := s(m["name"])
			id := s(h(h(a(m["instances"])[0])["attributes"])["id"])
			switch name {
			case "alice":
				aliceCID = id
			case "bob":
				bobCID = id
			default:
				t.Fatalf("invalid container name specified: %q", name)
			}
		}
	}

	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	aliceIP, err := waitForIP(ctx, docker, aliceCID)
	if err != nil {
		t.Fatal(err)
	}

	bobIP, err := waitForIP(ctx, docker, bobCID)
	if err != nil {
		t.Fatal(err)
	}

	for {
		time.Sleep(2 * waitTime)
		// this basically waits for both pings to succeed bofore the context above times out.
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		default:
		}

		if _, err := execContainer(ctx, docker, aliceCID, "/bin/ping", "-c", "1", bobIP.String()); err != nil {
			continue
		}

		if _, err := execContainer(ctx, docker, bobCID, "/bin/ping", "-c", "1", aliceIP.String()); err != nil {
			continue
		}
		break
	}
}
