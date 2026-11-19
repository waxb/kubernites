package model

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
)

const (
	QoSBestEffort = string(corev1.PodQOSBestEffort)
	QoSBurstable  = string(corev1.PodQOSBurstable)
	QoSGuaranteed = string(corev1.PodQOSGuaranteed)

	RankHigh   = "high"
	RankNormal = "normal"
	RankLow    = "low"
	rankPrefix = "realm-"
)

var Ranks = []string{RankHigh, RankNormal, RankLow}

type qosInfo struct {
	look, title, sentence string
	rank                  int
}

var qosTable = map[string]qosInfo{
	QoSBestEffort: {look: "vagrant", title: "Vagrant shack (BestEffort)", rank: 0,
		sentence: "No Pledge at all: among the first evicted when its Keep runs short."},
	QoSBurstable: {look: "freeman", title: "Freeman cottage (Burstable)", rank: 1,
		sentence: "Evicted early from a starved Keep only if it uses more than it pledged."},
	QoSGuaranteed: {look: "landed", title: "Landed manor (Guaranteed)", rank: 2,
		sentence: "Its Pledge equals its Binding: evicted last from a starved Keep, yet not immune."},
}

func QoSOf(p *corev1.Pod) string {
	if p.Status.QOSClass != "" {
		return string(p.Status.QOSClass)
	}
	return string(podutil.QoSClass(p))
}

func QoSLook(qos string) string { return qosTable[qos].look }

func QoSTitle(qos string) string { return qosTable[qos].title }

func QoSSentence(qos string) string { return qosTable[qos].sentence }

func QoSRank(qos string) int {
	if info, ok := qosTable[qos]; ok {
		return info.rank
	}
	return -1
}

func RankOf(priorityClass string) string {
	for _, r := range Ranks {
		if priorityClass == RankClass(r) {
			return r
		}
	}
	return ""
}

func RankClass(rank string) string { return rankPrefix + rank }

func priorityOf(p *corev1.Pod) int32 {
	if p.Spec.Priority == nil {
		return 0
	}
	return *p.Spec.Priority
}
