package kubectl

import (
	"strings"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

const hpaExplainPrefix = "horizontalpodautoscaler"

var hpaExplanations = buildHPAExplanations()

func hpaField(field, typ, text, lore string) explanation {
	return explanation{kind: "HorizontalPodAutoscaler", version: "autoscaling/v2", field: field, typ: typ, text: text, lore: lore}
}

func buildHPAExplanations() map[string]explanation {
	root := autoscalingv2.HorizontalPodAutoscaler{}.SwaggerDoc()
	spec := autoscalingv2.HorizontalPodAutoscalerSpec{}.SwaggerDoc()
	behavior := autoscalingv2.HorizontalPodAutoscalerBehavior{}.SwaggerDoc()
	rules := autoscalingv2.HPAScalingRules{}.SwaggerDoc()
	out := map[string]explanation{
		hpaExplainPrefix: {kind: "HorizontalPodAutoscaler", version: "autoscaling/v2", text: root[""],
			lore: "A War Horn: every 15 seconds the Stewards sound it, read the Omens of Augury (metrics-server) and call more huts to a crew when the food they eat runs past the target, or send them home when it falls. Without Augury, or if any worker pledges no Labor (requests.cpu), the Horn stays silent."},
		hpaExplainPrefix + ".spec": hpaField("spec", "HorizontalPodAutoscalerSpec", root["spec"],
			"How the War Horn is tuned: which crew it keeps, the lowest and highest count, the Omens it hears, and its temper."),
		hpaExplainPrefix + ".status": hpaField("status", "HorizontalPodAutoscalerStatus", root["status"],
			"What the War Horn last heard and did: current and desired count, the Omens it read, and its conditions (AbleToScale, ScalingActive, ScalingLimited)."),
		hpaExplainPrefix + ".spec.scaletargetref": hpaField("scaleTargetRef", "CrossVersionObjectReference", spec["scaleTargetRef"],
			"The crew the Horn keeps: an apps/v1 Deployment, or a bare ReplicaSet (a shift) of an older realm."),
		hpaExplainPrefix + ".spec.minreplicas": hpaField("minReplicas", "integer", spec["minReplicas"],
			"The lowest count the Horn leaves standing. Zero is allowed only for object or external Omens, never for food."),
		hpaExplainPrefix + ".spec.maxreplicas": hpaField("maxReplicas", "integer", spec["maxReplicas"],
			"The highest count the Horn may call. Each still needs quarters on a Keep."),
		hpaExplainPrefix + ".spec.metrics": hpaField("metrics", "[]MetricSpec", spec["metrics"],
			"The Omens the Horn hears. In this Age only food: resource cpu as a Utilization of the Pledge (70%) or an AverageValue (500m)."),
		hpaExplainPrefix + ".spec.behavior": hpaField("behavior", "HorizontalPodAutoscalerBehavior", spec["behavior"],
			"The Horn's temper. Left unset, it may call up to double the crew (and at least 4 huts) each sync, and keeps the highest call of the last five minutes before sending anyone home."),
	}
	for _, dir := range []string{"scaleUp", "scaleDown"} {
		base := hpaExplainPrefix + ".spec.behavior." + strings.ToLower(dir)
		out[base] = hpaField(dir, "HPAScalingRules", behavior[dir], directionLore[dir])
		out[base+".stabilizationwindowseconds"] = hpaField("stabilizationWindowSeconds", "integer", rules["stabilizationWindowSeconds"],
			"The Horn's memory: it looks back this many seconds and chooses the safest recent call.")
		out[base+".policies"] = hpaField("policies", "[]HPAScalingPolicy", rules["policies"],
			"Limits per period: Pods moves a fixed number of workers, Percent a share of the crew.")
		out[base+".selectpolicy"] = hpaField("selectPolicy", "string", rules["selectPolicy"],
			"Max picks the policy allowing the biggest change, Min the smallest, and Disabled silences this direction.")
		out[base+".tolerance"] = hpaField("tolerance", "Quantity", rules["tolerance"],
			"How far the Omens may stray from the target before the Horn stirs. Unset, the cluster-wide 10% applies.")
	}
	return out
}

var directionLore = map[string]string{
	"scaleUp":   "How eagerly the Horn calls huts: by default at once, up to double the crew or 4 more per 15 seconds, whichever is more.",
	"scaleDown": "How reluctantly the Horn sends workers home: by default it keeps the highest call of the last 300 seconds.",
}
