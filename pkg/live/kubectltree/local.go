package kubectltree

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/printers"
	sigyaml "sigs.k8s.io/yaml"

	"github.com/waxb/kubernites/pkg/live/write"
)

const ReasonClientSecret = "a client-side run prints a Secret only as YAML or JSON, with its values masked: use -o yaml or -o json"

var templateFormats = map[string]bool{
	"jsonpath": true, "jsonpath-file": true, "jsonpath-as-json": true, "go-template": true, "go-template-file": true,
	"template": true, "templatefile": true, "custom-columns": true, "custom-columns-file": true,
}

func templated(cmd *cobra.Command) bool {
	flags := cmd.Flags()
	output := ""
	if o := flags.Lookup("output"); o != nil {
		output, _, _ = strings.Cut(o.Value.String(), "=")
	}
	if templateFormats[output] {
		return true
	}
	t := flags.Lookup("template")
	return output == "" && t != nil && t.Value.String() != ""
}

var objectFlags = []string{"filename", "overrides"}

func makesSecrets(cmd *cobra.Command) bool {
	for _, name := range objectFlags {
		if f := cmd.Flags().Lookup(name); f != nil && f.Changed {
			return true
		}
	}
	for c := cmd; c != nil && c.Parent() != nil; c = c.Parent() {
		if c.Name() == "secret" && c.Parent().Name() == "create" {
			return true
		}
	}
	return false
}

const notMasked = "(a Secret, not shown: the Stone could not mask it)\n"

func printedFormat(cmd *cobra.Command) string {
	flags := cmd.Flags()
	output := ""
	if o := flags.Lookup("output"); o != nil {
		output, _, _ = strings.Cut(o.Value.String(), "=")
	}
	if t := flags.Lookup("template"); output == "" && t != nil && t.Value.String() != "" {
		output = "go-template"
	}
	return strings.ToLower(output)
}

func printedObjects(format, text string) ([]map[string]any, bool) {
	var docs [][]byte
	switch format {
	case "json":
		dec := json.NewDecoder(strings.NewReader(text))
		dec.UseNumber()
		var out []map[string]any
		for {
			var obj map[string]any
			err := dec.Decode(&obj)
			if errors.Is(err, io.EOF) {
				return out, true
			}
			if err != nil || obj == nil {
				return nil, false
			}
			out = append(out, obj)
		}
	case "yaml", "kyaml":
		r := yamlutil.NewYAMLReader(bufio.NewReader(strings.NewReader(text)))
		for {
			doc, err := r.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, false
			}
			if len(bytes.TrimSpace(doc)) > 0 && string(bytes.TrimSpace(doc)) != "---" {
				docs = append(docs, doc)
			}
		}
	default:
		return nil, false
	}
	var out []map[string]any
	for _, doc := range docs {
		data, err := sigyaml.YAMLToJSON(doc)
		if err != nil {
			return nil, false
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		var obj map[string]any
		if dec.Decode(&obj) != nil || obj == nil {
			return nil, false
		}
		out = append(out, obj)
	}
	return out, true
}

func printAgain(format string, objects []map[string]any) (string, bool) {
	var b bytes.Buffer
	for i, obj := range objects {
		switch format {
		case "json":
			out, err := json.MarshalIndent(obj, "", "    ")
			if err != nil {
				return "", false
			}
			b.Write(out)
			b.WriteByte('\n')
		case "yaml":
			if i > 0 {
				b.WriteString("---\n")
			}
			out, err := sigyaml.Marshal(obj)
			if err != nil {
				return "", false
			}
			b.Write(out)
		case "kyaml":
			if err := (&printers.KYAMLPrinter{}).PrintObj(&unstructured.Unstructured{Object: obj}, &b); err != nil {
				return "", false
			}
		}
	}
	return b.String(), true
}

func maskLocal(format, text string) string {
	if format != "json" && format != "yaml" && format != "kyaml" {
		return text
	}
	objects, ok := printedObjects(format, text)
	if !ok {
		if strings.Contains(text, "Secret") {
			return notMasked
		}
		return text
	}
	masked := false
	for _, obj := range objects {
		if write.MaskObject(obj) {
			masked = true
		}
	}
	if !masked {
		return text
	}
	out, ok := printAgain(format, objects)
	if !ok {
		return notMasked
	}
	return out
}
