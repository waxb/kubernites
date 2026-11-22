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

func folded(root, p string) bool {
	root, p = fold(root), fold(p)
	return p == root || within(root, p)
}

func fold(s string) string {
	return strings.Map(func(r rune) rune {
		least := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			least = min(least, f)
		}
		return least
	}, s)
}

const (
	ReasonProtected     = "the Stone's cp never writes ~/.kube, the kubeconfig chosen at connect or a credential file it names"
	ReasonProtectedRead = "the Stone's cp never sends ~/.kube, the kubeconfig chosen at connect or a credential file it names"
)

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
	if s.tree.protected(local) {
		s.run.refuse(ReasonProtectedRead)
		cmdutil.CheckErr(fmt.Errorf("%s: %s", ReasonProtectedRead, shown(local)))
	}
	s.run.localSide(local, false)
	read, skipped, err := s.copyToPod(pod, c.Name, local, dest.file, noPreserve)
	s.tellFiles("Read", local, read)
	for _, p := range skipped {
		fmt.Fprintf(s.ios.Out, "Skipped %s: %s\n", shown(p), ReasonProtectedRead)
	}
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

func (s *reach) copyToPod(pod *corev1.Pod, container, local, remote string, noPreserve bool) ([]string, []string, error) {
	if _, err := os.Lstat(local); err != nil {
		return nil, nil, fmt.Errorf("%s doesn't exist in local filesystem", local)
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
	var read, skipped []string
	tarred := make(chan error, 1)
	go func() {
		files, passed, err := makeTar(local, path.Base(destFile), writer, s.tree.protected)
		read, skipped = files, passed
		writer.CloseWithError(err)
		tarred <- err
	}()
	err := s.podExec(pod, container, cmdArr, reader, s.ios.Out)
	reader.CloseWithError(io.ErrClosedPipe)
	if tarErr := <-tarred; err == nil && tarErr != nil && !errors.Is(tarErr, io.ErrClosedPipe) {
		err = tarErr
	}
	return read, skipped, err
}

type tarWalk struct {
	tw      *tar.Writer
	read    []string
	skipped []string
	skip    func(string) bool
}

func makeTar(local, destName string, w io.Writer, skip func(string) bool) ([]string, []string, error) {
	tw := &tarWalk{tw: tar.NewWriter(w), skip: skip}
	src := filepath.Clean(local)
	matches := []string{src}
	if _, err := os.Lstat(src); err != nil {
		globbed, err := filepath.Glob(src)
		if err != nil {
			return nil, nil, err
		}
		matches = globbed
	}
	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("%s doesn't exist in local filesystem", local)
	}
	for _, m := range matches {
		name := destName
		if m != src {
			name = path.Join(destName, filepath.Base(m))
		}
		if err := tw.add(m, name); err != nil {
			return tw.read, tw.skipped, err
		}
	}
	return tw.read, tw.skipped, tw.tw.Close()
}

func (w *tarWalk) add(file, name string) error {
	if w.skip != nil && w.skip(file) {
		w.skipped = append(w.skipped, file)
		return nil
	}
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
			return w.tw.WriteHeader(hdr)
		}
		for _, e := range entries {
			if err := w.add(filepath.Join(file, e.Name()), path.Join(name, e.Name())); err != nil {
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
		return w.tw.WriteHeader(hdr)
	case !stat.Mode().IsRegular():
		return nil
	}
	hdr, err := tar.FileInfoHeader(stat, file)
	if err != nil {
		return err
	}
	hdr.Name = name
	if err := w.tw.WriteHeader(hdr); err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(w.tw, f); err != nil {
		return err
	}
	w.read = append(w.read, file)
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

const ReasonCopyEnded = "the copy stopped where it was answered N; what was written stays"

var (
	copyAskBytes  int64 = 1 << 30
	copyAskFiles        = 10000
	copyShowBytes int64 = 64 << 20
	copyShowFiles       = 1000
)

type copied struct {
	s         *reach
	dest      string
	bytes     int64
	files     int
	askBytes  int64
	askFiles  int
	showBytes int64
	showFiles int
}

func (c *copied) more(n int64, file bool) error {
	c.bytes += n
	if file {
		c.files++
	}
	if c.bytes >= c.showBytes || c.files >= c.showFiles {
		fmt.Fprintf(c.s.ios.ErrOut, "Copying: %s in %s so far\n", size(c.bytes), files(c.files))
		for c.bytes >= c.showBytes {
			c.showBytes += copyShowBytes
		}
		for c.files >= c.showFiles {
			c.showFiles += copyShowFiles
		}
	}
	if c.bytes < c.askBytes && c.files < c.askFiles {
		return nil
	}
	if !c.s.run.ask(c.s.ctx, Held{More: size(c.bytes) + " in " + files(c.files)}) {
		return errors.New(ReasonCopyEnded)
	}
	for c.bytes >= c.askBytes {
		c.askBytes += copyAskBytes
	}
	for c.files >= c.askFiles {
		c.askFiles += copyAskFiles
	}
	return nil
}

func (c *copied) write(out io.Writer, r io.Reader) error {
	for {
		step := min(c.askBytes-c.bytes, c.showBytes-c.bytes, 1<<20)
		n, err := io.CopyN(out, r, max(step, 1))
		if n > 0 {
			if merr := c.more(n, false); merr != nil {
				return merr
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func files(n int) string {
	if n == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", n)
}

func size(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

type madeSet struct {
	list []string
	seen map[string]bool
}

func (m *madeSet) add(p string) {
	if !m.seen[p] {
		m.seen[p] = true
		m.list = append(m.list, p)
	}
}

func (m *madeSet) drop(p string) {
	if m.seen[p] {
		delete(m.seen, p)
		m.list = slices.DeleteFunc(m.list, func(q string) bool { return q == p })
	}
}

func (s *reach) untarAll(prefix, dest string, r io.Reader) ([]string, error) {
	written := &madeSet{seen: map[string]bool{}}
	overwrite := 0
	count := &copied{s: s, dest: dest, askBytes: copyAskBytes, askFiles: copyAskFiles, showBytes: copyShowBytes, showFiles: copyShowFiles}
	tr := tar.NewReader(r)
	dest = filepath.Clean(dest)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return written.list, nil
		}
		if err != nil {
			return written.list, err
		}
		if !strings.HasPrefix(hdr.Name, prefix) {
			return written.list, errors.New("tar contents corrupted")
		}
		if err := s.untarOne(hdr, tr, prefix, dest, written, &overwrite, count); err != nil {
			return written.list, err
		}
		if err := count.more(0, true); err != nil {
			return written.list, err
		}
	}
}

func (s *reach) untarOne(hdr *tar.Header, tr io.Reader, prefix, dest string, written *madeSet, overwrite *int, count *copied) error {
	target := filepath.Join(dest, filepath.FromSlash(hdr.Name[len(prefix):]))
	if !within(dest, target) && target != dest {
		fmt.Fprintf(s.ios.ErrOut, "warning: file %q is outside target destination, skipping\n", target)
		return nil
	}
	if throughLink(dest, target) {
		fmt.Fprintf(s.ios.ErrOut, "warning: %q would be written through a link, skipping\n", target)
		return nil
	}
	if s.tree.protected(target) {
		fmt.Fprintf(s.ios.ErrOut, "warning: %q is ~/.kube, the kubeconfig chosen at connect or a credential file it names, skipping\n", target)
		return nil
	}
	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := s.makeDirs(target, written); err != nil {
			return err
		}
		if s.tree.protected(target) {
			s.unmake(target, written)
			fmt.Fprintf(s.ios.ErrOut, "warning: %s is ~/.kube, the kubeconfig chosen at connect or a credential file it names, skipping\n", shown(target))
		}
		return nil
	case tar.TypeSymlink:
		fmt.Fprintf(s.ios.ErrOut, "warning: skipping symlink: %q -> %q\n", target, hdr.Linkname)
		return nil
	case tar.TypeReg:
	default:
		fmt.Fprintf(s.ios.ErrOut, "warning: skipping %q: not a regular file\n", target)
		return nil
	}
	exists := false
	if _, err := os.Lstat(target); err == nil {
		exists = true
		if *overwrite == 0 {
			*overwrite = -1
			if s.run.ask(s.ctx, Held{Overwrite: shown(target)}) {
				*overwrite = 1
			}
		}
		if *overwrite < 0 {
			fmt.Fprintf(s.ios.ErrOut, "Kept %s: it already exists on this computer\n", shown(target))
			return nil
		}
	}
	if err := s.makeDirs(filepath.Dir(target), written); err != nil {
		return err
	}
	if s.tree.protected(target) {
		s.unmake(filepath.Dir(target), written)
		fmt.Fprintf(s.ios.ErrOut, "warning: %s is ~/.kube, the kubeconfig chosen at connect or a credential file it names, skipping\n", shown(target))
		return nil
	}
	if exists {
		if err := os.Remove(target); err != nil {
			return err
		}
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	written.add(target)
	if err := errors.Join(count.write(out, tr), out.Close()); err != nil {
		written.drop(target)
		os.Remove(target)
		fmt.Fprintf(s.ios.ErrOut, "Not copied: %s, cut short while it was written; the part written was removed\n", shown(target))
		return err
	}
	return nil
}

func (s *reach) unmake(dir string, written *madeSet) {
	for cur := filepath.Clean(dir); ; cur = filepath.Dir(cur) {
		d := cur + string(filepath.Separator)
		if !written.seen[d] || os.Remove(cur) != nil {
			return
		}
		written.drop(d)
	}
}

func (s *reach) makeDirs(dir string, written *madeSet) error {
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
		written.add(made[i] + string(filepath.Separator))
	}
	return nil
}
