package lexicon

import "testing"

func TestNamesModeLeavesNoStoryWordFromTheCaptureReview(t *testing.T) {
	cases := map[string]string{
		"the realm's reserve sends one Stone Keep":   "the cluster's reserve sends one xlarge Node",
		"A Stone Keep the reserve has added":         "An xlarge Node the reserve has added",
		"the next Stone Keep arrives in 420 seconds": "the next xlarge Node arrives in 420 seconds",
		"Stone Keep 2": "xlarge Node 2",
		"STONE KEEP":   "XLARGE NODE",
		"Raise a Great Keep beside the Citadel Keeps": "Raise a 2xlarge Node beside the 4xlarge Nodes",
		"CITADEL KEEP":                   "4XLARGE NODE",
		"Parcels Depot":                  "parcels Service",
		"PARCELS DEPOT":                  "parcels SERVICE",
		"Parcels Depot: Depot (parcels)": "parcels Service: Service (web requests)",
		"at 4, a ship sails in with more boxes than 4 can carry":                         "at 4, a request burst arrives with more requests than 4 can carry",
		"A bigger ship docks: word of your porters has spread, and its cogs now sail in": "A bigger request burst arrives: word of your porters has spread, and its small bursts now arrive",
		"its ledgers sail in once clerks carry them":                                     "its database requests arrive once clerks carry them",
		"A large ship docks at the harbor":                                               "A large request burst arrives at the harbor",
		"$ kubectl get keeps":                                                            "$ kubectl get nodes",
		"GROUND 18%":                                                                     "DISK 18%",
	}
	for story, want := range cases {
		if got := TrueNames(story); got != want {
			t.Errorf("TrueNames(%q) = %q, want %q", story, got, want)
		}
	}
}
