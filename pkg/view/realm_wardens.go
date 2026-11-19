package view

import (
	"fmt"
)

const ActPlaceWarden = "placeWarden"

const ActRaiseDaemonSet = "raiseDaemonSet"

const WardenImage = "warden:1.0"

const WardenApp = "warden"

func PlaceWardenCommand(keep string) string {
	return fmt.Sprintf(`kubectl run %s --image=%s --labels=app=%s --overrides='{"spec":{"nodeSelector":{"kubernetes.io/hostname":"%s"}}}'`,
		WardenPodName(keep), WardenImage, WardenApp, keep)
}

func WardenPodName(keep string) string {
	return WardenApp + "-" + keep
}
