package live

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	scopeNamespaced = "Namespaced"
	metadataWatch   = "application/json;as=PartialObjectMetadata;g=meta.k8s.io;v=v1"
)

var errNoMetadata = errors.New("this connection cannot read metadata alone")

type Gateway struct {
	metav1.ObjectMeta `json:"metadata"`
	Spec              GatewaySpec `json:"spec"`
}

type GatewaySpec struct {
	GatewayClassName string            `json:"gatewayClassName,omitempty"`
	Listeners        []GatewayListener `json:"listeners,omitempty"`
}

type GatewayListener struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname,omitempty"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

type CRD struct {
	Name    string
	UID     types.UID
	Group   string
	Kind    string
	Plural  string
	Scope   string
	Version string
}

type CustomResource struct {
	Group     string
	Kind      string
	Namespace string `json:",omitempty"`
	Name      string
	UID       types.UID
}

type crdObject struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
	Spec     struct {
		Group string `json:"group"`
		Names struct {
			Kind   string `json:"kind"`
			Plural string `json:"plural"`
		} `json:"names"`
		Scope    string `json:"scope"`
		Versions []struct {
			Name    string `json:"name"`
			Served  bool   `json:"served"`
			Storage bool   `json:"storage"`
		} `json:"versions"`
	} `json:"spec"`
}

func (o crdObject) served() string {
	best := ""
	for _, v := range o.Spec.Versions {
		switch {
		case !v.Served:
		case v.Storage:
			return v.Name
		case best == "":
			best = v.Name
		}
	}
	return best
}

func readCRDs(ctx context.Context, r *reading, path string) error {
	items, version, err := listAt[crdObject](ctx, r.l.getJSON, path, r.page, r.since)
	r.version = version
	r.s.CRDs = nil
	for _, o := range items {
		if c, ok := o.crd(); ok {
			r.s.CRDs = append(r.s.CRDs, c)
		}
	}
	sortCRDs(r.s.CRDs)
	return err
}

func sortCRDs(list []CRD) {
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
}

func (o crdObject) crd() (CRD, bool) {
	v := o.served()
	if v == "" || o.Spec.Names.Plural == "" {
		return CRD{}, false
	}
	return CRD{Name: o.Metadata.Name, UID: o.Metadata.UID, Group: o.Spec.Group, Kind: o.Spec.Names.Kind,
		Plural: o.Spec.Names.Plural, Scope: o.Spec.Scope, Version: v}, true
}

func feedCRD(s *Snapshot, typ string, raw []byte) (metav1.Object, error) {
	var o crdObject
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, err
	}
	s.CRDs = slices.DeleteFunc(s.CRDs, func(c CRD) bool { return c.UID == o.Metadata.UID })
	if c, ok := o.crd(); ok && typ != watchDeleted {
		s.CRDs = append(s.CRDs, c)
		sortCRDs(s.CRDs)
	}
	s.forgetCustom()
	return &o.Metadata, nil
}

func (s *Snapshot) forgetCustom() {
	served := map[string]bool{}
	for _, c := range s.CRDs {
		served[c.Group+"/"+c.Kind] = true
		served[c.List()] = true
	}
	s.Custom = slices.DeleteFunc(s.Custom, func(r CustomResource) bool { return !served[r.Group+"/"+r.Kind] })
	gone := func(list string) bool { return strings.Contains(list, ".") && !served[list] }
	s.Denied = slices.DeleteFunc(s.Denied, gone)
	s.Missing = slices.DeleteFunc(s.Missing, gone)
	s.Refused = slices.DeleteFunc(s.Refused, func(r Refusal) bool { return gone(r.List) })
	for list := range s.Lists {
		if gone(list) {
			delete(s.Lists, list)
		}
	}
}

var crdKind = kindRead{name: "customresourcedefinitions", root: "/apis/apiextensions.k8s.io/v1", read: readCRDs, feed: feedCRD,
	items: func(s *Snapshot) []metav1.Object {
		out := make([]metav1.Object, len(s.CRDs))
		for i, c := range s.CRDs {
			out[i] = &metav1.ObjectMeta{Name: c.Name, UID: c.UID}
		}
		return out
	},
	swap: func(dst, src *Snapshot) {
		dst.CRDs = src.CRDs
		dst.forgetCustom()
	}}

func (c CRD) List() string {
	return c.Plural + "." + c.Group
}

func (c CRD) native() bool {
	root := "/apis/" + c.Group + "/" + c.Version
	for _, k := range kindReads {
		if k.root == root && k.name == c.Plural {
			return true
		}
	}
	return false
}

func (c CRD) owns(r CustomResource) bool {
	return r.Group == c.Group && r.Kind == c.Kind
}

func (s *Snapshot) serves(c CRD) bool {
	return slices.Contains(s.CRDs, c)
}

func customKind(c CRD) kindRead {
	return kindRead{name: c.List(), root: "/apis/" + url.PathEscape(c.Group) + "/" + url.PathEscape(c.Version), resource: c.Plural,
		namespaced: c.Scope == scopeNamespaced, custom: true, accept: metadataWatch,
		read: func(ctx context.Context, r *reading, path string) error {
			meta, ok := r.l.(metaLister)
			if !ok {
				return errNoMetadata
			}
			items, version, err := listAt[metav1.PartialObjectMetadata](ctx, meta.getMeta, path, r.page, r.since)
			r.version = version
			r.s.Custom = slices.DeleteFunc(r.s.Custom, c.owns)
			for _, it := range items {
				r.s.Custom = append(r.s.Custom, CustomResource{Group: c.Group, Kind: c.Kind, Namespace: it.Namespace, Name: it.Name, UID: it.UID})
			}
			return err
		},
		feed: func(s *Snapshot, typ string, raw []byte) (metav1.Object, error) {
			var o metav1.PartialObjectMetadata
			if err := json.Unmarshal(raw, &o); err != nil {
				return nil, err
			}
			if !s.serves(c) {
				return &o.ObjectMeta, nil
			}
			at := slices.IndexFunc(s.Custom, func(r CustomResource) bool { return r.UID == o.UID })
			res := CustomResource{Group: c.Group, Kind: c.Kind, Namespace: o.Namespace, Name: o.Name, UID: o.UID}
			switch {
			case typ == watchDeleted && at >= 0:
				s.Custom = append(s.Custom[:at], s.Custom[at+1:]...)
			case typ == watchDeleted:
			case at >= 0:
				s.Custom[at] = res
			default:
				s.Custom = append(s.Custom, res)
			}
			return &o.ObjectMeta, nil
		},
		items: func(s *Snapshot) []metav1.Object {
			var out []metav1.Object
			for _, r := range s.Custom {
				if c.owns(r) {
					out = append(out, &metav1.ObjectMeta{Name: r.Name, Namespace: r.Namespace, UID: r.UID})
				}
			}
			return out
		},
		swap: func(dst, src *Snapshot) {
			if !dst.serves(c) {
				return
			}
			dst.Custom = slices.DeleteFunc(dst.Custom, c.owns)
			for _, r := range src.Custom {
				if c.owns(r) {
					dst.Custom = append(dst.Custom, r)
				}
			}
		}}
}

func (r *reading) readCustom(ctx context.Context) {
	if _, ok := r.l.(metaLister); !ok || len(r.s.CRDs) == 0 {
		return
	}
	r.at.Total += len(r.s.CRDs)
	for _, c := range r.s.CRDs {
		if c.native() {
			r.at.Done++
			continue
		}
		k := customKind(c)
		r.at.List, r.at.Count = k.name, 0
		r.tell()
		r.version = ""
		err := k.read(ctx, r, k.path(""))
		ns := ""
		if IsForbidden(err) && k.namespaced {
			refusal := Refusal{List: k.name, Message: messageOf(err)}
			r.version = ""
			err = k.read(ctx, r, k.path(r.ns))
			if err == nil {
				refusal.Namespace, ns = r.ns, r.ns
			}
			r.s.Refused = append(r.s.Refused, refusal)
		}
		switch {
		case err == nil:
			r.mark(k.name, ns)
		case IsForbidden(err):
			r.s.Denied = append(r.s.Denied, k.name)
		default:
			r.s.Missing = append(r.s.Missing, k.name)
		}
		r.at.Done++
		r.tell()
	}
}
