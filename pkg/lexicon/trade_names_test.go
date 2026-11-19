package lexicon

import "testing"

func TestATradeReadsAsAnImageInNamesMode(t *testing.T) {
	for story, want := range map[string]string{
		"new trades and the outer isles still lie ahead.":                 "new images and the outer zones still lie ahead.",
		"A shift can be raised now, to keep a count of one trade's huts.": "A ReplicaSet can be raised now, to keep a count of one image's Pods.",
		"Every class of this trade, with its size":                        "Every tag of this image, with its size",
		"The previous class of the trade":                                 "The previous tag of the image",
		"Porters, class of 1.1: Taught to lift with their legs":           "porter Pods, tag 1.1: Taught to lift with their legs",
		"More ships from the west now land at the harbor.":                "More request bursts from the west now land at the harbor.",
		"Grow your trade: build, then let crews carry for you":            "Grow your trade: build, then let Deployments carry for you",
		"A realm that serves well is talked about, and more trade comes.": "A cluster that serves well is talked about, and more trade comes.",
		"each pulled from the registry in the trade its image names.":     "each pulled from the registry in the image it names.",
		"kubectl run <name> --image=<trade:class> [--labels=k=v,...]":     "kubectl run <name> --image=<image:tag> [--labels=k=v,...]",
	} {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) =\n%q, want\n%q", story, got, want)
		}
	}
}

func TestABadgeVerbReadsAsLabelInNamesMode(t *testing.T) {
	if got, want := TrueNames("Badge it for study (debug=true)"), "Label it for study (debug=true)"; got != want {
		t.Fatalf("TrueNames = %q, want %q", got, want)
	}
}

func TestAHarborPortReadsAsAGatewayInNamesMode(t *testing.T) {
	for story, want := range map[string]string{
		"Upgrade a port for caravels":                     "Upgrade a Gateway for medium bursts",
		"open a berth or a port.":                         "open a traffic ramp or a Gateway.",
		"Deeper ports bring new kinds of box.":            "Deeper Gateways bring new kinds of request.",
		"Draws more ships to this port.":                  "Draws more request bursts to this Gateway.",
		"kubectl expose deployment clerks --port=80":      "kubectl expose deployment clerks --port=80",
		"ClusterIP PORT 80 -> 8080":                       "ClusterIP PORT 80 -> 8080",
		"Host Port":                                       "Host Port",
		"on the control-plane Node's hostPort 80":         "on the control-plane Node's hostPort 80",
		"the Service's port 80 sends to the target port.": "the Service's port 80 sends to the target port.",
	} {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) =\n%q, want\n%q", story, got, want)
		}
	}
}

func TestATreasureChestStaysABonusInNamesMode(t *testing.T) {
	if got, want := TrueNames("The first ship's treasure chest opens: +250 rations"), "The first request burst's bonus opens: +250m of CPU"; got != want {
		t.Fatalf("TrueNames = %q, want %q", got, want)
	}
	if got, want := TrueNames("Hauls crates and chests"), "Hauls uploads and large uploads"; got != want {
		t.Fatalf("TrueNames = %q, want %q", got, want)
	}
}

func TestAFullQuayReadsAsAFullQueueInNamesMode(t *testing.T) {
	if got, want := TrueNames("Quay full 1.2/s: scale porters to 4"), "Queue full 1.2/s: scale porters to 4"; got != want {
		t.Fatalf("TrueNames = %q, want %q", got, want)
	}
}

func TestADecreeVerbReadsAsDeclareInNamesMode(t *testing.T) {
	for story, want := range map[string]string{
		"The Steward, the old builder of realms, decrees: restart the control plane, four halls that let your realm run itself.": "The controller, the old builder of clusters, declares: restart the control plane, four components that let your cluster run itself.",
		"He decrees: restart the realm's control plane, the four halls that let your realm run itself.":                          "He declares: restart the cluster's control plane, the four components that let your cluster run itself.",
		"You can now give orders at the Speaking Stone, and every hall works through the Chancery.":                              "You can now give orders at kubectl, and every component works through the kube-apiserver.",
		"He decrees again when a later step needs it.":                                                                           "He declares again when a later step needs it.",
		"write decrees, kubectl apply": "write manifests, kubectl apply",
	} {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) =\n%q, want\n%q", story, got, want)
		}
	}
}

func TestAHireCounterCountsRunPodsInNamesMode(t *testing.T) {
	for story, want := range map[string]string{
		"ready: press HIRE (hired 1/3)":          "ready: press RUN (run 1/3)",
		"struck 1/2, hired 0/2, on the roll 1/2": "struck 1/2, run 0/2, on the EndpointSlice 1/2",
	} {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) = %q, want %q", story, got, want)
		}
	}
}

func TestOneNineReadsAsOneSLOWithoutAnEcho(t *testing.T) {
	if got, want := TrueNames("Availability: One Nine (90%) over a full window"), "Availability: a 90% SLO over a full window"; got != want {
		t.Fatalf("TrueNames = %q, want %q", got, want)
	}
}

func TestRoomsCountMemoryInMebibytesInNamesMode(t *testing.T) {
	for story, want := range map[string]string{
		"housing in rooms":    "memory in mebibytes",
		"64 rooms of housing": "64Mi of memory",
	} {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) = %q, want %q", story, got, want)
		}
	}
}

func TestAClassReadsAsATagInNamesMode(t *testing.T) {
	for story, want := range map[string]string{
		"Each trade climbs through its classes: pull a class once, and every hut of that trade can take it.": "Each image climbs through its tags: pull a tag once, and every Pod of that image can take it.",
		"Bring the clerks to class 3.0: the haulers come then.":                                              "Bring the clerks to tag 3.0: the haulers come then.",
		"QoS class: Burstable":                  "QoS class: Burstable",
		"its quality class is Guaranteed":       "its quality class is Guaranteed",
		"the storage class vault-wrights":       "the storage class vault-wrights",
		"Ingress market/accounts (class nginx)": "Ingress market/accounts (class nginx)",
	} {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) =\n%q, want\n%q", story, got, want)
		}
	}
}

func TestTheRoadsReforgeColumnsReadRebuildInNamesMode(t *testing.T) {
	for story, want := range map[string]string{"A REFORGE KEEPS": "A REBUILD KEEPS", "A REFORGE LOSES": "A REBUILD LOSES"} {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) = %q, want %q", story, got, want)
		}
	}
}

func TestTheStoneInTheSquareReadsAsTheStoneInNamesMode(t *testing.T) {
	if got, want := TrueNames("Press Enter, or click the Stone in the market square."), "Press Enter, or click the kubectl stone in the market square."; got != want {
		t.Fatalf("TrueNames = %q, want %q", got, want)
	}
}
