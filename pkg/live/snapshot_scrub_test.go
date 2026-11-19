package live

import (
	"os"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestARecordingKeepsNoProxyVariableOfTheMachineThatRecordedIt(t *testing.T) {
	pod := corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Env: []corev1.EnvVar{
		{Name: "HTTPS_PROXY", Value: "http://127.0.0.1:3128"}, {Name: "no_proxy", Value: "example.internal"}, {Name: "LOG_LEVEL", Value: "2"}}}}}}
	s := Snapshot{Pods: []corev1.Pod{pod}}
	got := s.WithoutProxyEnvironment().Pods[0].Spec.Containers[0].Env
	if len(got) != 1 || got[0].Name != "LOG_LEVEL" {
		t.Errorf("the scrubbed env is %v, want only LOG_LEVEL", got)
	}
	if len(s.Pods[0].Spec.Containers[0].Env) != 3 {
		t.Error("scrubbing changed the snapshot it was given")
	}
	for _, path := range []string{"testdata/kind-snapshot.json", "testdata/stone-kind-snapshot.json"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ToUpper(string(data))
		for _, v := range []string{`"HTTPS_PROXY"`, `"HTTP_PROXY"`, `"NO_PROXY"`, `"ALL_PROXY"`} {
			if strings.Contains(text, v) {
				t.Errorf("%s holds %s from the machine that recorded it", path, v)
			}
		}
	}
}
