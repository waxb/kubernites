package lexicon

import "testing"

func TestTrueNamesGuardVerbsUnitsAndIdentifiers(t *testing.T) {
	cases := map[string]string{
		"Keeps your declarations true, count by count":                 "Keeps your declarations true, count by count",
		"FOR  Keeps the realm as you declared it":                      "FOR  Keeps the cluster as you declared it",
		"For your cluster: Keeps a count of porters":                   "For your cluster: Keeps a count of porters",
		"Keeps every object you declare":                               "Keeps every object you declare",
		"Keeps what you declare true":                                  "Keeps what you declare true",
		"Keep stone-keep-1 is raised":                                  "Node stone-keep-1 is raised",
		"Retire a small Keep the proper way":                           "Retire a small Node the proper way",
		"Keeps join NotReady":                                          "Nodes join NotReady",
		"EACH ARCHIVIST KEEPS ITS OWN DEED, WHATEVER KEEP IT LANDS ON": "EACH ARCHIVE POD KEEPS ITS OWN PERSISTENTVOLUMECLAIM, WHATEVER NODE IT LANDS ON",
		"The Reeve runs it from its own shelf":                         "The kubelet runs it from its manifest folder",
		"runs it from a scroll on its own shelf":                       "runs it from its manifest folder",
		"A Vault of 10 shelves":                                        "A PersistentVolume of 10 Gi",
		"A Vault claims shelves":                                       "A PersistentVolume claims storage",
		"service/parcels exposed":                                      "service/parcels exposed",
		"deployment/porters scaled":                                    "deployment/porters scaled",
		"Open the parcels depot":                                       "Open the parcels Service",
		"PARCELS DEPOT":                                                "parcels SERVICE",
		"Each worker pledges 2,000 rations of food":                    "Each container requests 2 CPUs",
		"8 rooms of housing":                                           "8Mi of memory",
		"The Nines are counted":                                        "The SLI is counted",
		"The Nines now ask 99%":                                        "The SLI now asks 99%",
		"when the Nines hold":                                          "when the SLI holds",
		"the [[nines|Nines]] count against you":                        "the [[nines|SLI]] counts against you",
		"Hold the Nine first":                                          "Hold the SLO first",
		"Nine in ten boxes":                                            "Nine in ten requests",
		"are BestEffort Vagrants":                                      "are BestEffort",
		"the largest Keep the Sky Barons rent":                         "the largest Node the cloud provider rents",
		"Raise a Great Keep after the Stone Keep":                      "Raise a 2xlarge Node after the xlarge Node",
	}
	for in, want := range cases {
		if got := TrueNames(in); got != want {
			t.Errorf("TrueNames(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}
