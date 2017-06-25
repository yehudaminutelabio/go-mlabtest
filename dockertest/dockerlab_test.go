package dockertest

import (
	"fmt"
	"testing"

	"net/http"
)

func TestDocker(t *testing.T) {
	docker, _ := New(t, "", nil)
	defer docker.Close()

	err := ping(docker)
	if err != nil {
		t.Fatal("DockerLab ping failed:", err)
	}

	t.Log("DockerLab ping OK")
}

// Ping mlab
func ping(p *DockerLab) error {
	ip, port, err := p.GetAddressPort()
	resp, err := http.Get(fmt.Sprintf("http://%s:%d/_ping", ip.String(), port))
	if err != nil {
		return err
	}

	resp.Body.Close()
	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("Error code %d", resp.StatusCode)

}
