package view

import (
	"fmt"
)

const ParcelPile = "pile:parcel"

func FamiliarCommand(crew, name, image string) string {
	return fmt.Sprintf(`kubectl patch deployment %s -p '{"spec":{"template":{"spec":{"initContainers":[{"name":"%s","image":"%s","restartPolicy":"Always"}]}}}}'`, crew, name, image)
}
