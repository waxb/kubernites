package kubectl

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	explainWrap     = 76
	explainFieldPad = "  "
	explainTextPad  = "    "
)

var kindTypes = map[string]reflect.Type{
	"Pod": reflect.TypeFor[corev1.Pod](), "Node": reflect.TypeFor[corev1.Node](), "Service": reflect.TypeFor[corev1.Service](),
	"Event": reflect.TypeFor[corev1.Event](), "Namespace": reflect.TypeFor[corev1.Namespace](), "ConfigMap": reflect.TypeFor[corev1.ConfigMap](),
	"Deployment": reflect.TypeFor[appsv1.Deployment](), "ReplicaSet": reflect.TypeFor[appsv1.ReplicaSet](),
	"EndpointSlice":           reflect.TypeFor[discoveryv1.EndpointSlice](),
	"HorizontalPodAutoscaler": reflect.TypeFor[autoscalingv2.HorizontalPodAutoscaler](),
	"PriorityClass":           reflect.TypeFor[schedulingv1.PriorityClass](),
	"LimitRange":              reflect.TypeFor[corev1.LimitRange](),
	"DaemonSet":               reflect.TypeFor[appsv1.DaemonSet](),
	"ControllerRevision":      reflect.TypeFor[appsv1.ControllerRevision](),
	"StatefulSet":             reflect.TypeFor[appsv1.StatefulSet](),
	"ServiceAccount":          reflect.TypeFor[corev1.ServiceAccount](),
	"PersistentVolume":        reflect.TypeFor[corev1.PersistentVolume](),
	"PersistentVolumeClaim":   reflect.TypeFor[corev1.PersistentVolumeClaim](),
	"StorageClass":            reflect.TypeFor[storagev1.StorageClass](),
}

var stringTypes = map[reflect.Type]bool{reflect.TypeFor[metav1.Time](): true, reflect.TypeFor[metav1.MicroTime](): true}

var namedTypes = map[reflect.Type]string{reflect.TypeFor[resource.Quantity](): "Quantity", reflect.TypeFor[intstr.IntOrString](): "IntOrString"}

type apiField struct {
	name     string
	typ      reflect.Type
	doc      string
	required bool
}

func structOf(t reflect.Type) (reflect.Type, bool) {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
		if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
			return nil, false
		}
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || stringTypes[t] || namedTypes[t] != "" {
		return nil, false
	}
	return t, true
}

func swaggerDoc(t reflect.Type) map[string]string {
	m, ok := t.MethodByName("SwaggerDoc")
	if !ok || m.Type.NumIn() != 1 || m.Type.NumOut() != 1 {
		return nil
	}
	out := m.Func.Call([]reflect.Value{reflect.New(t).Elem()})
	doc, _ := out[0].Interface().(map[string]string)
	return doc
}

func jsonName(f reflect.StructField) (string, bool, bool) {
	tag := f.Tag.Get("json")
	name, opts, _ := strings.Cut(tag, ",")
	if name == "-" || !f.IsExported() {
		return "", false, false
	}
	return name, strings.Contains(opts, "inline") || (name == "" && f.Anonymous), !strings.Contains(opts, "omitempty")
}

func apiFields(t reflect.Type) []apiField {
	doc := swaggerDoc(t)
	out := []apiField{}
	for i := range t.NumField() {
		f := t.Field(i)
		name, inline, required := jsonName(f)
		switch {
		case inline:
			out = append(out, apiFields(f.Type)...)
		case name != "":
			out = append(out, apiField{name: name, typ: f.Type, doc: doc[name], required: required})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func fieldTypeName(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case stringTypes[t]:
		return "string"
	case namedTypes[t] != "":
		return namedTypes[t]
	}
	switch t.Kind() {
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return "string"
		}
		return "[]" + fieldTypeName(t.Elem())
	case reflect.Map:
		return "map[string]" + fieldTypeName(t.Elem())
	case reflect.Struct:
		return t.Name()
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	}
	return "string"
}

func fieldAt(kind string, path []string) (reflect.Type, bool) {
	t, ok := kindTypes[kind]
	for _, seg := range path {
		st, isStruct := structOf(t)
		if !ok || !isStruct {
			return nil, false
		}
		ok = false
		for _, f := range apiFields(st) {
			if strings.EqualFold(f.name, seg) {
				t, ok = f.typ, true
				break
			}
		}
	}
	return t, ok
}

func typeExplanation(r ResourceType, path []string) (explanation, error) {
	t, ok := kindTypes[r.Kind]
	if !ok {
		return explanation{}, fmt.Errorf("error: the server doesn't have a resource type %q", r.Plural)
	}
	e := explanation{kind: r.Kind, version: r.apiVersion, text: swaggerDoc(t)[""]}
	for _, seg := range path {
		st, isStruct := structOf(t)
		found := false
		for _, f := range apiFields(st) {
			if isStruct && strings.EqualFold(f.name, seg) {
				t, found = f.typ, true
				e.field, e.typ, e.text = f.name, fieldTypeName(f.typ), f.doc
				break
			}
		}
		if !found {
			return explanation{}, fmt.Errorf("error: field %q does not exist", seg)
		}
	}
	if e.text == "" {
		e.text = "<empty>"
	}
	return e, nil
}

func fieldList(kind string, path []string) string {
	t, ok := fieldAt(kind, path)
	if !ok {
		return ""
	}
	st, isStruct := structOf(t)
	if !isStruct {
		return ""
	}
	var b strings.Builder
	for _, f := range apiFields(st) {
		b.WriteString(explainFieldPad + f.name + "\t<" + fieldTypeName(f.typ) + ">")
		if f.required {
			b.WriteString(" -required-")
		}
		doc := f.doc
		if doc == "" {
			doc = "<no description>"
		}
		b.WriteString("\n" + wrapIndented(doc, explainWrap, explainTextPad) + "\n\n")
	}
	return b.String()
}

func wrapIndented(text string, width int, pad string) string {
	lines := []string{}
	for _, para := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			if line != "" && len(line)+1+len(word) > width {
				lines = append(lines, strings.TrimRight(pad+line, " "))
				line = ""
			}
			if line != "" {
				line += " "
			}
			line += word
		}
		lines = append(lines, strings.TrimRight(pad+line, " "))
	}
	return strings.Join(lines, "\n")
}
