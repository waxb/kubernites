package model

import (
	"maps"

	discoveryv1 "k8s.io/api/discovery/v1"
)

const (
	userNamespace = "default"
	apiService    = "kubernetes"
)

type GuildRoll struct {
	Service   string
	Namespace string
	Ready     int
	Endpoints int
	Members   []Member
}

type Member struct {
	Name    string
	Ready   bool
	Address string
}

type NodeView struct {
	Name           string
	Unschedulable  bool
	Taints         []string
	SystemCPUMilli int64
	SystemMemBytes int64
	Labels         map[string]string
	TaintSpecs     []string
}

func (c *Cache) NodeViews() map[string]NodeView {
	out := map[string]NodeView{}
	shares := systemShares(c.Pods())
	for _, n := range c.Nodes() {
		held := shares[n.Name]
		v := NodeView{Name: n.Name, Unschedulable: n.Spec.Unschedulable, SystemCPUMilli: held.cpu, SystemMemBytes: held.mem}
		for _, t := range n.Spec.Taints {
			v.Taints = append(v.Taints, t.Key)
		}
		v.Labels, v.TaintSpecs = maps.Clone(n.Labels), TaintSpecs(n.Spec.Taints)
		out[n.Name] = v
	}
	return out
}

func (c *Cache) GuildRolls() []GuildRoll {
	return c.guildRolls(func(ns string) bool { return ns == userNamespace })
}

func (c *Cache) EveryGuildRoll() []GuildRoll {
	return c.guildRolls(func(string) bool { return true })
}

func (c *Cache) guildRolls(kept func(ns string) bool) []GuildRoll {
	counts := map[string]GuildRoll{}
	listed := map[string]bool{}
	for _, s := range c.Slices() {
		if !kept(s.Namespace) {
			continue
		}
		name := s.Namespace + "/" + s.Labels[discoveryv1.LabelServiceName]
		roll := counts[name]
		for _, e := range s.Endpoints {
			key := name + "/" + endpointKey(e)
			if listed[key] {
				continue
			}
			listed[key] = true
			ready := e.Conditions.Ready == nil || *e.Conditions.Ready
			roll.Endpoints++
			if ready {
				roll.Ready++
			}
			roll.Members = append(roll.Members, Member{Name: memberName(e), Ready: ready, Address: firstAddress(e)})
		}
		counts[name] = roll
	}
	out := []GuildRoll{}
	for _, svc := range c.Services() {
		if !kept(svc.Namespace) || svc.Name == apiService && svc.Namespace == userNamespace {
			continue
		}
		roll := counts[svc.Namespace+"/"+svc.Name]
		roll.Service, roll.Namespace = svc.Name, svc.Namespace
		out = append(out, roll)
	}
	return out
}

func endpointKey(e discoveryv1.Endpoint) string {
	if len(e.Addresses) > 0 {
		return e.Addresses[0]
	}
	return memberName(e)
}

func firstAddress(e discoveryv1.Endpoint) string {
	if len(e.Addresses) == 0 {
		return ""
	}
	return e.Addresses[0]
}

func memberName(e discoveryv1.Endpoint) string {
	switch {
	case e.TargetRef != nil:
		return e.TargetRef.Name
	case len(e.Addresses) > 0:
		return e.Addresses[0]
	}
	return "?"
}
