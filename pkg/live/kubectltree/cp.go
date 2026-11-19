package kubectltree

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/cmd/util/podcmd"
	"k8s.io/kubectl/pkg/scheme"
)

const ReasonRetries = "the Stone's cp does not resume a broken copy: copy again"

type fileSpec struct {
	namespace string
	pod       string
	file      string
}

var errFileSpec = errors.New("filespec must match the canonical format: [[namespace/]pod:]file/path")

func extractFileSpec(arg string) (fileSpec, error) {
	i := strings.Index(arg, ":")
	switch {
	case i == 0:
		return fileSpec{}, errFileSpec
	case i < 0:
		return fileSpec{file: arg}, nil
	}
	pod, file := arg[:i], arg[i+1:]
	pieces := strings.Split(pod, "/")
	switch len(pieces) {
	case 1:
		return fileSpec{pod: pieces[0], file: file}, nil
	case 2:
		return fileSpec{namespace: pieces[0], pod: pieces[1], file: file}, nil
	}
	return fileSpec{}, errFileSpec
}

func (t *Tree) LocalPath(p string) string {
	switch {
	case p == "~":
		return filepath.Clean(t.local)
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(t.local, p[2:])
	case filepath.IsAbs(p):
		return filepath.Clean(p)
	}
	return filepath.Join(t.LocalDir(), p)
}

func (t *Tree) LocalDir() string {
	return filepath.Join(t.local, "Downloads", "Kubernites")
}

func (t *Tree) protected(p string) bool {
	kube := filepath.Join(t.local, ".kube")
	p = filepath.Clean(p)
	if p == kube || within(kube, p) {
		return true
	}
	kubeInfo, kubeErr := os.Stat(kube)
	var configs []os.FileInfo
	for _, k := range t.paths {
		if st, err := os.Stat(k); err == nil {
			configs = append(configs, st)
		}
		if filepath.Clean(k) == p {
			return true
		}
	}
	for cur := p; ; cur = filepath.Dir(cur) {
		if st, err := os.Stat(cur); err == nil {
			if kubeErr == nil && os.SameFile(st, kubeInfo) {
				return true
			}
			for _, c := range configs {
				if os.SameFile(st, c) {
					return true
				}
			}
		}
		if filepath.Dir(cur) == cur {
			return false
		}
	}
}

const ReasonProtected = "the Stone's cp never writes ~/.kube or the kubeconfig chosen at connect"

func (s *reach) cpRun(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		cmdutil.CheckErr(errors.New("source and destination are required"))
	}
	src, err := extractFileSpec(args[0])
	cmdutil.CheckErr(err)
	dest, err := extractFileSpec(args[1])
	cmdutil.CheckErr(err)
	switch {
	case src.pod != "" && dest.pod != "":
		cmdutil.CheckErr(errors.New("one of src or dest must be a local file specification"))
	case src.file == "" || dest.file == "":
		cmdutil.CheckErr(errors.New("filepath can not be empty"))
	case src.pod == "" && dest.pod == "":
		cmdutil.CheckErr(errors.New("one of src or dest must be a remote file specification"))
	}
	ns, _, err := s.f.ToRawKubeConfigLoader().Namespace()
	cmdutil.CheckErr(err)
	remote := src
	if remote.pod == "" {
		remote = dest
	}
	if remote.namespace == "" {
		remote.namespace = ns
	}
	cs, err := s.f.KubernetesClientSet()
	cmdutil.CheckErr(err)
	pod, err := cs.CoreV1().Pods(remote.namespace).Get(s.ctx, remote.pod, metav1.GetOptions{})
	cmdutil.CheckErr(err)
	container := flagString(cmd, "container")
	c, err := podcmd.FindOrDefaultContainerByName(pod, container, false, s.ios.ErrOut)
	cmdutil.CheckErr(err)
	noPreserve := flagBool(cmd, "no-preserve")
	if src.pod != "" {
		local := s.tree.LocalPath(dest.file)
		if s.tree.protected(local) {
			s.run.refuse(ReasonProtected)
			cmdutil.CheckErr(fmt.Errorf("%s: %s", ReasonProtected, local))
		}
		s.run.localSide(local, true)
		written, err := s.copyFromPod(pod, c.Name, src.file, local)
		s.tellFiles("Wrote", local, written)
		cmdutil.CheckErr(err)
		return
	}
	local := s.tree.LocalPath(src.file)
	s.run.localSide(local, false)
	read, err := s.copyToPod(pod, c.Name, local, dest.file, noPreserve)
	s.tellFiles("Read", local, read)
	cmdutil.CheckErr(err)
}

func shown(p string) string {
	if strings.IndexFunc(p, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
		return strconv.Quote(p)
	}
	return p
}

func (s *reach) tellFiles(verb, root string, files []string) {
	var dirs, plain []string
	for _, f := range files {
		if d, ok := strings.CutSuffix(f, string(filepath.Separator)); ok {
			dirs = append(dirs, d)
		} else {
			plain = append(plain, f)
		}
	}
	switch {
	case len(plain) == 1 && plain[0] == root:
		fmt.Fprintf(s.ios.Out, "%s %s\n", verb, shown(root))
	case len(plain) > 0:
		noun := "files"
		if len(plain) == 1 {
			noun = "file"
		}
		fmt.Fprintf(s.ios.Out, "%s %d %s under %s:\n", verb, len(plain), noun, shown(root))
		for _, f := range plain {
			fmt.Fprintf(s.ios.Out, "  %s\n", shown(f))
		}
	}
	for _, d := range dirs {
		fmt.Fprintf(s.ios.Out, "Made the directory %s\n", shown(d))
	}
}

func (s *reach) podExec(pod *corev1.Pod, container string, command []string, stdin io.Reader, stdout io.Writer) error {
	client, err := s.f.RESTClient()
	if err != nil {
		return err
	}
	cfg, err := s.f.ToRESTConfig()
	if err != nil {
		return err
	}
	req := client.Get().Resource("pods").Name(pod.Name).Namespace(pod.Namespace).SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{Container: container, Command: command, Stdin: stdin != nil, Stdout: true, Stderr: true}, scheme.ParameterCodec)
	u := req.URL()
	session := s.begin(u)
	session.Verb = "cp"
	err = s.executor(u, cfg, s.ctx, remotecommand.StreamOptions{Stdin: stdin, Stdout: stdout, Stderr: s.ios.ErrOut})
	s.finish(session, nil, err)
	return err
}

func (s *reach) copyFromPod(pod *corev1.Pod, container, remote, local string) ([]string, error) {
	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := s.podExec(pod, container, []string{"tar", "cf", "-", remote}, nil, writer)
		writer.CloseWithError(err)
		done <- err
	}()
	prefix := stripPathShortcuts(path.Clean(strings.TrimLeft(remote, "/")))
	written, err := s.untarAll(prefix, local, reader)
	reader.CloseWithError(io.ErrClosedPipe)
	if execErr := <-done; err == nil {
		err = execErr
	}
	return written, err
}

func (s *reach) copyToPod(pod *corev1.Pod, container, local, remote string, noPreserve bool) ([]string, error) {
	if _, err := os.Lstat(local); err != nil {
		return nil, fmt.Errorf("%s doesn't exist in local filesystem", local)
	}
	destFile := path.Clean(remote)
	if s.podExec(pod, container, []string{"test", "-d", remote}, nil, io.Discard) == nil {
		destFile = path.Join(destFile, filepath.Base(local))
	}
	cmdArr := []string{"tar", "-xmf", "-"}
	if noPreserve {
		cmdArr = []string{"tar", "--no-same-permissions", "--no-same-owner", "-xmf", "-"}
	}
	if dir := path.Dir(destFile); dir != "" {
		cmdArr = append(cmdArr, "-C", dir)
	}
	reader, writer := io.Pipe()
	var read []string
	tarred := make(chan error, 1)
	go func() {
		files, err := makeTar(local, path.Base(destFile), writer)
		read = files
		writer.CloseWithError(err)
		tarred <- err
	}()
	err := s.podExec(pod, container, cmdArr, reader, s.ios.Out)
	reader.CloseWithError(io.ErrClosedPipe)
	if tarErr := <-tarred; err == nil && tarErr != nil && !errors.Is(tarErr, io.ErrClosedPipe) {
		err = tarErr
	}
	return read, err
}

func makeTar(local, destName string, w io.Writer) ([]string, error) {
	tw := tar.NewWriter(w)
	var read []string
	src := filepath.Clean(local)
	matches := []string{src}
	if _, err := os.Lstat(src); err != nil {
		globbed, err := filepath.Glob(src)
		if err != nil {
			return nil, err
		}
		matches = globbed
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("%s doesn't exist in local filesystem", local)
	}
	for _, m := range matches {
		name := destName
		if m != src {
			name = path.Join(destName, filepath.Base(m))
		}
		if err := tarPath(m, name, tw, &read); err != nil {
			return read, err
		}
	}
	return read, tw.Close()
}

func tarPath(file, name string, tw *tar.Writer, read *[]string) error {
	stat, err := os.Lstat(file)
	if err != nil {
		return err
	}
	switch {
	case stat.IsDir():
		entries, err := os.ReadDir(file)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			hdr, _ := tar.FileInfoHeader(stat, file)
			hdr.Name = name
			return tw.WriteHeader(hdr)
		}
		for _, e := range entries {
			if err := tarPath(filepath.Join(file, e.Name()), path.Join(name, e.Name()), tw, read); err != nil {
				return err
			}
		}
		return nil
	case stat.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(file)
		if err != nil {
			return err
		}
		hdr, _ := tar.FileInfoHeader(stat, target)
		hdr.Name, hdr.Linkname = name, target
		return tw.WriteHeader(hdr)
	case !stat.Mode().IsRegular():
		return nil
	}
	hdr, err := tar.FileInfoHeader(stat, file)
	if err != nil {
		return err
	}
	hdr.Name = name
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(tw, f); err != nil {
		return err
	}
	*read = append(*read, file)
	return f.Close()
}

func stripPathShortcuts(p string) string {
	newPath := p
	trimmed := strings.TrimPrefix(newPath, "../")
	for trimmed != newPath {
		newPath = trimmed
		trimmed = strings.TrimPrefix(newPath, "../")
	}
	if newPath == "." || newPath == ".." {
		newPath = ""
	}
	if len(newPath) > 0 && string(newPath[0]) == "/" {
		return newPath[1:]
	}
	return newPath
}

func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func throughLink(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return true
	}
	cur := root
	if rel == "." {
		st, err := os.Lstat(p)
		return err == nil && st.Mode()&os.ModeSymlink != 0
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		cur = filepath.Join(cur, part)
		st, err := os.Lstat(cur)
		if errors.Is(err, os.ErrNotExist) {
			return false
		}
		if err != nil || st.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func (s *reach) untarAll(prefix, dest string, r io.Reader) ([]string, error) {
	var written []string
	overwrite := 0
	tr := tar.NewReader(r)
	dest = filepath.Clean(dest)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return written, nil
		}
		if err != nil {
			return written, err
		}
		if !strings.HasPrefix(hdr.Name, prefix) {
			return written, errors.New("tar contents corrupted")
		}
		target := filepath.Join(dest, filepath.FromSlash(hdr.Name[len(prefix):]))
		if !within(dest, target) && target != dest {
			fmt.Fprintf(s.ios.ErrOut, "warning: file %q is outside target destination, skipping\n", target)
			continue
		}
		if throughLink(dest, target) {
			fmt.Fprintf(s.ios.ErrOut, "warning: %q would be written through a link, skipping\n", target)
			continue
		}
		if s.tree.protected(target) {
			fmt.Fprintf(s.ios.ErrOut, "warning: %q is ~/.kube or the kubeconfig chosen at connect, skipping\n", target)
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := s.makeDirs(target, &written); err != nil {
				return written, err
			}
			continue
		case tar.TypeSymlink:
			fmt.Fprintf(s.ios.ErrOut, "warning: skipping symlink: %q -> %q\n", target, hdr.Linkname)
			continue
		case tar.TypeReg:
		default:
			fmt.Fprintf(s.ios.ErrOut, "warning: skipping %q: not a regular file\n", target)
			continue
		}
		if _, err := os.Lstat(target); err == nil {
			if overwrite == 0 {
				overwrite = -1
				if s.run.ask(s.ctx, Held{Overwrite: target}) {
					overwrite = 1
				}
			}
			if overwrite < 0 {
				fmt.Fprintf(s.ios.ErrOut, "Kept %s: it already exists on this computer\n", target)
				continue
			}
		}
		if err := s.makeDirs(filepath.Dir(target), &written); err != nil {
			return written, err
		}
		if !slices.Contains(written, target) {
			written = append(written, target)
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return written, err
		}
		_, cerr := io.Copy(out, tr)
		if err := errors.Join(cerr, out.Close()); err != nil {
			return written, err
		}
	}
}

func (s *reach) makeDirs(dir string, written *[]string) error {
	var made []string
	for cur := filepath.Clean(dir); ; cur = filepath.Dir(cur) {
		if _, err := os.Lstat(cur); err == nil || filepath.Dir(cur) == cur {
			break
		}
		made = append(made, cur)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i := len(made) - 1; i >= 0; i-- {
		if d := made[i] + string(filepath.Separator); !slices.Contains(*written, d) {
			*written = append(*written, d)
		}
	}
	return nil
}
