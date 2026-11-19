package model

import "fmt"

type ObjRef struct {
	Kind      string
	Namespace string
	Name      string
}

type Autonomy struct {
	Kind   string
	Actor  string
	Text   string
	At     float64
	Object ObjRef
}

func (a Autonomy) Key() string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%.3f", a.Kind, a.Actor, a.Object.Kind, a.Object.Namespace, a.Object.Name, a.At)
}
