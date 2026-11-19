package lexicon

import "testing"

func TestAJSONPatchIsNoStoryCommand(t *testing.T) {
	if StoryCommandShown(`kubectl patch endpointslice parcels-by-hand --type=json -p '[{"op":"add","path":"/endpoints/-","value":{"addresses":["10.244.0.3"]}}]'`) {
		t.Fatal("story mode offers a raw JSON patch to type")
	}
	if StoryCommandShown(`kubectl patch deployment porters -p '{"spec":{"template":{"spec":{"containers":[{"name":"porter","livenessProbe":{"httpGet":{"path":"/healthz","port":8080}}}]}}}}'`) {
		t.Fatal("story mode offers a raw strategic merge patch to type")
	}
	if !StoryCommandShown("kubectl delete endpointslice parcels-by-hand") || !StoryCommandShown("kubectl set selector service parcels app=porters") {
		t.Fatal("story mode hides a plain command")
	}
}
