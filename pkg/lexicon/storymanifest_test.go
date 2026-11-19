package lexicon

import (
	"reflect"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

const crewManifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: porters
spec:
  replicas: 3
  selector:
    matchLabels:
      app: porters
  template:
    metadata:
      labels:
        app: porters
    spec:
      containers:
      - name: porter
        image: porter:1.0
        resources:
          requests:
            cpu: 200m
            memory: 64Mi
          limits:
            memory: 128Mi`

const crewStory = `book: crews/v1
decree: crew
about:
  name: porters
plan:
  count: 3
  looksFor:
    badges:
      app: porters
  template:
    about:
      badges:
        app: porters
    plan:
      workers:
      - name: porter
        trade: porter:1.0
        resources:
          pledge:
            food: 200
            housing: 64
          binding:
            housing: 128`

var scriptoriumManifests = []string{crewManifest, `apiVersion: v1
kind: Pod
metadata:
  name: porter-declared
  labels:
    app: porters
spec:
  containers:
  - name: porter
    image: porter:1.0`, `apiVersion: v1
kind: Service
metadata:
  name: parcels
spec:
  selector:
    app: porters
  ports:
  - port: 80
    targetPort: 8080`, `apiVersion: v1
kind: ConfigMap
metadata:
  name: realm-charter
data:
  motto: only what is declared survives`, `apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: porters
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: porters
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70`}

func TestAManifestReadsInTheRealmsWords(t *testing.T) {
	if got := StoryManifest(crewManifest); got != crewStory {
		t.Fatalf("story manifest:\n%s\nwant\n%s", got, crewStory)
	}
	if got := StoryText(crewManifest); got != crewStory {
		t.Fatalf("StoryText leaves a manifest in Kubernetes words:\n%s", got)
	}
	if got := StoryText(crewStory); got != crewStory {
		t.Fatalf("StoryText changed a story manifest:\n%s", got)
	}
}

func TestAStoryManifestReadsBackExactly(t *testing.T) {
	for _, m := range scriptoriumManifests {
		story := StoryManifest(m)
		if !IsStoryManifest(story) {
			t.Fatalf("not recognised as a story manifest:\n%s", story)
		}
		if KubernetesWords.MatchString(story) {
			t.Errorf("story manifest keeps a Kubernetes word %q:\n%s", KubernetesWords.FindString(story), story)
		}
		if back := ManifestFromStory(story); back != m {
			t.Errorf("round trip:\n%s\nwant\n%s", back, m)
		}
	}
}

func TestStoryUnitsReadBackAsTheSameQuantity(t *testing.T) {
	m := strings.Replace(crewManifest, "memory: 128Mi", "memory: 1Gi", 1)
	m = strings.Replace(m, "cpu: 200m", "cpu: \"0.5\"", 1)
	back := ManifestFromStory(StoryManifest(m))
	var want, got map[string]any
	if err := yaml.Unmarshal([]byte(strings.NewReplacer("1Gi", "1024Mi", `"0.5"`, "500m").Replace(m)), &want); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal([]byte(back), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("quantities changed:\n%s", back)
	}
}

func TestAPatchEchoAndItsProbeFieldsSpeakStoryWords(t *testing.T) {
	cmd := `kubectl patch deployment porters -p '{"spec":{"template":{"spec":{"containers":[{"name":"porter","readinessProbe":{"httpGet":{"path":"/ready","port":8080},"periodSeconds":10,"failureThreshold":3,"timeoutSeconds":1}}]}}}}'`
	story := StoryText(cmd)
	for _, old := range []string{"readinessProbe", "httpGet", "periodSeconds", "failureThreshold", "timeoutSeconds", "containers", `"template"`} {
		if strings.Contains(story, old) {
			t.Errorf("the echo keeps %q: %s", old, story)
		}
	}
	for _, want := range []string{`"shutters":`, `"ask":`, `"everySeconds":`, `"missesAllowed":`, `"patienceSeconds":`, `"workers":`} {
		if !strings.Contains(story, want) {
			t.Errorf("the echo lacks %s: %s", want, story)
		}
	}
	if back := CommandFromStory(story); back != cmd {
		t.Errorf("typing the echo back does not run the real patch: %s", back)
	}
}

func TestAStoryManifestSpeaksTheShuttersAndTheBell(t *testing.T) {
	kube := "spec:\n  containers:\n  - name: porter\n    readinessProbe:\n      httpGet:\n        path: /ready\n        port: 8080\n      periodSeconds: 10\n    livenessProbe:\n      failureThreshold: 3\n"
	story := StoryManifest(kube)
	for _, want := range []string{"shutters:", "ask:", "everySeconds:", "bell:", "missesAllowed:"} {
		if !strings.Contains(story, want) {
			t.Errorf("story manifest lacks %q:\n%s", want, story)
		}
	}
	if back := ManifestFromStory(story); back != kube {
		t.Errorf("round trip differs:\n%s\nwant\n%s", back, kube)
	}
}

func TestOnlyTheTopLevelStatusBecomesTheReport(t *testing.T) {
	kube := "apiVersion: apps/v1\nkind: Deployment\nstatus:\n  conditions:\n  - type: Available\n    status: \"True\"\n    message: Deployment has minimum availability.\n"
	story := StoryManifest(kube)
	if !strings.Contains(story, "report:\n") || !strings.Contains(story, "    status: \"True\"") {
		t.Errorf("only the top-level status is the report:\n%s", story)
	}
	if strings.Contains(story, "Deployment") {
		t.Errorf("a message keeps a Kubernetes word:\n%s", story)
	}
	if back := ManifestFromStory(strings.Replace(story, "Crew has minimum availability.", "Deployment has minimum availability.", 1)); back != kube {
		t.Errorf("round trip differs:\n%s", back)
	}
}
