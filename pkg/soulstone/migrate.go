package soulstone

import "strings"

var retiredRepositories = map[string]string{"lantern": "porter", "scribe": "clerk"}

func splitRepository(image string) (lead, repo, rest string) {
	body := image
	if after, ok := strings.CutPrefix(image, RegistryHost+"/"); ok {
		lead, body = RegistryHost+"/", after
	}
	end := len(body)
	if at := strings.IndexAny(body, ":@"); at >= 0 {
		end = at
	}
	return lead, body[:end], body[end:]
}

func IsRetired(image string) bool {
	_, repo, _ := splitRepository(image)
	_, retired := retiredRepositories[repo]
	return retired
}

func Migrate(image string) string {
	lead, repo, rest := splitRepository(image)
	if next, ok := retiredRepositories[repo]; ok {
		return lead + next + rest
	}
	return image
}
