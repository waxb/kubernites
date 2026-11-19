package kubectltree

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/yaml"
)

type Edit struct {
	Group           string
	Version         string
	Kind            string
	Resource        string
	Namespace       string
	Name            string
	ResourceVersion string
	Format          string
	Text            string
}

func (e Edit) Ref() string {
	ref := e.Resource
	if e.Group != "" {
		ref += "." + e.Group
	}
	return ref + "/" + e.Name
}

type editAsk struct {
	args        []string
	files       []string
	recursive   bool
	format      string
	managed     bool
	subresource string
}

func routeEdit(c *cobra.Command, r *run) {
	for _, name := range []string{"field-manager", "output-patch", "save-config"} {
		lockFlag(c.Flags().Lookup(name), ReasonManager, r)
	}
	c.PreRun, c.PreRunE, c.RunE = nil, nil, nil
	c.Run = func(cmd *cobra.Command, args []string) {
		ask := &editAsk{args: append([]string{}, args...)}
		ask.files, _ = cmd.Flags().GetStringSlice("filename")
		ask.recursive, _ = cmd.Flags().GetBool("recursive")
		ask.format, _ = cmd.Flags().GetString("output")
		ask.managed, _ = cmd.Flags().GetBool("show-managed-fields")
		ask.subresource, _ = cmd.Flags().GetString("subresource")
		switch ask.format {
		case "", "yaml":
			ask.format = "yaml"
		case "json":
		default:
			cmdutil.CheckErr(fmt.Errorf("the Scriptorium edits yaml or json, not %q", ask.format))
		}
		if len(ask.args) == 0 && len(ask.files) == 0 {
			cmdutil.CheckErr(cmdutil.UsageErrorf(cmd, "you must specify one object to edit"))
		}
		r.mu.Lock()
		r.edit = ask
		r.mu.Unlock()
	}
}

func fetchEdit(cf *genericclioptions.ConfigFlags, ask *editAsk, errOut io.Writer) (*Edit, int) {
	e, err := readEdit(cf, ask)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return nil, 1
	}
	return e, 0
}

func readEdit(cf *genericclioptions.ConfigFlags, ask *editAsk) (*Edit, error) {
	f := cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(cf))
	ns, enforce, err := f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}
	b := f.NewBuilder().Unstructured().NamespaceParam(ns).DefaultNamespace().
		FilenameParam(enforce, &resource.FilenameOptions{Filenames: ask.files, Recursive: ask.recursive}).
		ResourceTypeOrNameArgs(true, ask.args...).Latest().Flatten()
	if ask.subresource != "" {
		b = b.Subresource(ask.subresource)
	}
	infos, err := b.Do().Infos()
	if err != nil {
		return nil, err
	}
	switch {
	case len(infos) == 0:
		return nil, errors.New("edit cancelled, no objects found")
	case len(infos) > 1:
		return nil, fmt.Errorf("the Scriptorium opens one object at a time, and this names %d", len(infos))
	}
	info := infos[0]
	u, ok := info.Object.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("%s cannot be written out", info.ObjectName())
	}
	obj := u.DeepCopy()
	if !ask.managed {
		obj.SetManagedFields(nil)
	}
	var text []byte
	if ask.format == "json" {
		text, err = json.MarshalIndent(obj.Object, "", "    ")
		text = append(text, '\n')
	} else {
		text, err = yaml.Marshal(obj.Object)
	}
	if err != nil {
		return nil, err
	}
	gvk := obj.GroupVersionKind()
	return &Edit{
		Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind, Resource: info.Mapping.Resource.Resource,
		Namespace: obj.GetNamespace(), Name: obj.GetName(), ResourceVersion: obj.GetResourceVersion(),
		Format: ask.format, Text: string(text),
	}, nil
}
