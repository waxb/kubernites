package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

type treeRealm interface {
	Tree() *kubectltree.Tree
	Gate() *write.Gate
	Snapshot() live.Snapshot
}

func liveTreeOf(r any) (treeRealm, bool) {
	t, ok := r.(treeRealm)
	if !ok || t.Tree() == nil {
		return nil, false
	}
	return t, true
}

func (s *Session) treeBackend(b hud.Backend) hud.Backend {
	t, ok := liveTreeOf(s.Realm)
	if !ok {
		return b
	}
	b.Start = func(cmd string) (hud.StoneJob, bool) {
		j := s.stoneJob("kubectl " + strings.TrimPrefix(strings.TrimSpace(cmd), "kubectl "))
		j.screen = newStoneScreen(s.Hud.TerminalCells())
		return j.start(t, cmd), true
	}
	b.Send = func(e hud.LiveEdit, text string, force bool) (hud.StoneJob, bool) {
		return s.stoneJob(scriptoriumOrder(e, force)).send(t, e, text, force), true
	}
	b.Reveal = func(e hud.LiveEdit, key string) <-chan hud.Revealed {
		out := make(chan hud.Revealed, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), revealTimeout)
			defer cancel()
			v, err := t.Tree().Reveal(ctx, e.Namespace, e.Name, key)
			out <- hud.Revealed{Value: v, OK: err == nil}
		}()
		return out
	}
	return b
}

const revealTimeout = 5 * time.Second

type answerTo struct {
	id  int
	yes bool
}

type treeJob struct {
	job        *kubectltree.Job
	screen     *stoneScreen
	asking     sync.Mutex
	mu         sync.Mutex
	ask        *hud.StoneAsk
	asked      int
	reply      chan answerTo
	fail       *hud.StoneEnd
	order      string
	record     func(order string, w kubectltree.Write)
	session    func(order string, s kubectltree.Session)
	passed     func(write.Class)
	prep       chan struct{}
	cancelPrep context.CancelFunc
	stopped    bool
}

func newTreeJob() *treeJob {
	return &treeJob{reply: make(chan answerTo, 1)}
}

func (s *Session) stoneJob(order string) *treeJob {
	j := newTreeJob()
	j.order, j.record, j.session = order, s.recordStoneWrite, s.recordSession
	if w := s.Writer; w != nil && w.pass != nil {
		j.passed = func(c write.Class) { w.later(func(*Session) { w.pass(c) }) }
	}
	return j
}

func scriptoriumOrder(e hud.LiveEdit, force bool) string {
	order := "kubectl apply --server-side --field-manager=" + kubectltree.FieldManager
	if force {
		order += " --force-conflicts"
	}
	ref := strings.ToLower(e.Kind) + "/" + e.Name
	if e.Namespace != "" {
		ref += " -n " + e.Namespace
	}
	return order + " (the Scriptorium's " + ref + ")"
}

func StartTree(t treeRealm, cmd string) hud.StoneJob {
	return newTreeJob().start(t, cmd)
}

func (j *treeJob) start(t treeRealm, cmd string) *treeJob {
	if j.screen == nil {
		j.job = t.Tree().Start(cmd, j.hold(t))
		return j
	}
	j.job = t.Tree().StartIn(cmd, j.hold(t), j.screen.term)
	return j
}

func (j *treeJob) Screen() hud.StoneScreen {
	if j.screen == nil {
		return nil
	}
	return j.screen
}

func SendTree(t treeRealm, e hud.LiveEdit, text string, force bool) hud.StoneJob {
	return newTreeJob().send(t, e, text, force)
}

func (j *treeJob) send(t treeRealm, e hud.LiveEdit, text string, force bool) *treeJob {
	if len(maskedKeys(e.Kind, text)) == 0 {
		j.job = t.Tree().StartApply(text, force, j.hold(t))
		return j
	}
	ctx, cancel := context.WithCancel(context.Background())
	prep := make(chan struct{})
	j.mu.Lock()
	j.prep, j.cancelPrep = prep, cancel
	j.mu.Unlock()
	go func() {
		defer close(prep)
		defer cancel()
		restored, err := unmaskSend(ctx, t, e, text)
		j.mu.Lock()
		defer j.mu.Unlock()
		if j.stopped {
			j.fail = &hud.StoneEnd{Stopped: true, Sent: hud.StoneEditEnd{Output: ui.Bridge("STOP ended the send before it was sent: nothing was sent.", "Stopped before the apply: nothing was sent.")}}
			return
		}
		if err != nil {
			j.fail = &hud.StoneEnd{Sent: hud.StoneEditEnd{Output: ui.Bridge("A hidden value could not be read again, so nothing was sent: ", "A masked value could not be read again, so nothing was sent: ") + err.Error()}}
			return
		}
		j.job = t.Tree().StartApply(restored, force, j.hold(t))
	}()
	return j
}

func (j *treeJob) current() (*kubectltree.Job, *hud.StoneEnd, bool) {
	j.mu.Lock()
	prep := j.prep
	j.mu.Unlock()
	if prep != nil {
		select {
		case <-prep:
		default:
			return nil, nil, false
		}
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.job, j.fail, true
}

var unmaskSend = unmask

func unmask(parent context.Context, t treeRealm, e hud.LiveEdit, text string) (string, error) {
	for _, key := range maskedKeys(e.Kind, text) {
		ctx, cancel := context.WithTimeout(parent, revealTimeout)
		v, err := t.Tree().Reveal(ctx, e.Namespace, e.Name, key)
		cancel()
		if err != nil {
			return "", err
		}
		next, ok := hud.RevealLine(text, key, v)
		if !ok {
			return "", fmt.Errorf("the key %s was not found in the Scriptorium", key)
		}
		text = next
	}
	return text, nil
}

func (j *treeJob) hold(t treeRealm) kubectltree.Hold {
	gate, server := t.Gate(), t.Snapshot().Context.Server
	return func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, gate, server, h, func(a live.StoneAsk) bool { return j.wait(ctx, a) })
	}
}

func (j *treeJob) wait(ctx context.Context, a live.StoneAsk) bool {
	j.asking.Lock()
	defer j.asking.Unlock()
	ask := RenderAsk(a)
	j.mu.Lock()
	j.asked++
	ask.ID = j.asked
	j.ask = &ask
	j.mu.Unlock()
	defer func() {
		j.mu.Lock()
		if j.ask != nil && j.ask.ID == ask.ID {
			j.ask = nil
		}
		j.mu.Unlock()
	}()
	for {
		select {
		case r := <-j.reply:
			if r.id != ask.ID {
				continue
			}
			yes := r.yes && ask.Confirm
			if yes && a.Why == write.NeedsClass && j.passed != nil {
				j.passed(a.Class)
			}
			return yes
		case <-ctx.Done():
			return false
		}
	}
}

func (j *treeJob) Take() ([]hud.StoneChunk, bool) {
	job, _, ready := j.current()
	if !ready {
		return nil, false
	}
	if job == nil {
		return nil, true
	}
	chunks, done := job.Take()
	if j.record != nil {
		for _, w := range job.TakeWrites() {
			j.record(j.order, w)
		}
	}
	out := make([]hud.StoneChunk, len(chunks))
	for i, c := range chunks {
		out[i] = hud.StoneChunk{Text: c.Text, Err: c.Stderr}
	}
	return out, done
}

func (j *treeJob) Ask() (hud.StoneAsk, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.ask == nil {
		return hud.StoneAsk{}, false
	}
	return *j.ask, true
}

func (j *treeJob) Answer(yes bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.ask == nil {
		return
	}
	id := j.ask.ID
	j.ask = nil
	select {
	case <-j.reply:
	default:
	}
	j.reply <- answerTo{id: id, yes: yes}
}

func (j *treeJob) Stop() {
	j.Answer(false)
	j.mu.Lock()
	j.stopped = true
	cancel, job := j.cancelPrep, j.job
	j.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if job != nil {
		job.Stop()
	}
}

func (j *treeJob) End() hud.StoneEnd {
	job, fail, _ := j.current()
	if fail != nil {
		return *fail
	}
	if job == nil {
		return hud.StoneEnd{}
	}
	res := job.Result()
	if j.screen != nil {
		j.screen.close()
	}
	end := hud.StoneEnd{OK: res.OK(), Stopped: res.Stopped}
	for _, ss := range res.Sessions {
		end.Sessions = append(end.Sessions, stoneSession(ss))
		if j.session != nil {
			j.session(j.order, ss)
		}
	}
	if e := res.Edit; e != nil {
		end.Edit = &hud.LiveEdit{Ref: e.Ref(), Kind: e.Kind, Namespace: e.Namespace, Name: e.Name, Format: e.Format, Text: e.Text, Hidden: maskedKeys(e.Kind, e.Text)}
	}
	if a, ok := job.Applied(); ok {
		end.Sent = hud.StoneEditEnd{OK: a.OK(), Output: strings.TrimSpace(a.Output), Stale: a.Stale, Refused: strings.Contains(a.Output, live.ErrReadOnly.Error())}
		for _, c := range a.Conflicts {
			manager := c.Manager
			if c.Subresource != "" {
				manager += " (" + c.Subresource + ")"
			}
			end.Sent.Conflicts = append(end.Sent.Conflicts, hud.EditConflict{Manager: manager, Fields: c.Fields})
		}
	}
	return end
}

func maskedKeys(kind, text string) []string {
	if kind != "Secret" {
		return nil
	}
	var obj struct {
		APIVersion string            `json:"apiVersion"`
		Kind       string            `json:"kind"`
		Data       map[string]string `json:"data"`
	}
	if yaml.Unmarshal([]byte(text), &obj) != nil || obj.APIVersion != "v1" || obj.Kind != "Secret" {
		return nil
	}
	var keys []string
	for k, v := range obj.Data {
		if v == hud.MaskedValue {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}

const maxAskBody = 240

func RenderAsk(a live.StoneAsk) hud.StoneAsk {
	if a.More != "" {
		return hud.StoneAsk{Title: "KEEP COPYING ONTO THIS COMPUTER?", Lines: []string{
			"This copy has written " + a.More + " under " + a.Local + ".",
			"Y copies on and asks again at the next 1 GiB or 10,000 entries; N stops here and keeps what was written whole.",
		}, Confirm: true, Go: "COPY ON [Y]", GoTip: "Copy on; the Stone asks again at the next 1 GiB or 10,000 entries"}
	}
	if a.Overwrite != "" {
		return hud.StoneAsk{Title: "OVERWRITE A FILE ON THIS COMPUTER?", Lines: []string{
			a.Overwrite + " already exists on this computer.",
			"Y replaces it and every other file this copy finds already there; N keeps them all.",
		}, Confirm: true, Go: "OVERWRITE [Y]", GoTip: "Replace the files this copy finds already on this computer"}
	}
	if a.Why == write.NeedsClass {
		text := classText(a.Class)
		if a.Class == write.Tokens {
			text = []string{tokenWarning(a.Level)}
		}
		lines := append(text, ui.Bridge("Passed once, it is remembered for this realm until RESET LIVE CHOICES.", "Passed once, it is remembered for this context until RESET LIVE CHOICES."))
		return hud.StoneAsk{Title: "! " + strings.ToUpper(classTitle(a.Class)) + " !", Lines: lines, Confirm: true, Go: "I UNDERSTAND [Y]", GoTip: "Pass this warning once for this context; it is remembered until RESET LIVE CHOICES"}
	}
	if a.Connect {
		return hud.StoneAsk{Title: ui.Bridge("CONFIRM: open this connection into the hut?", "CONFIRM: "+a.Command), Lines: openingLines(a), Confirm: true}
	}
	r := a.Order.Request
	path := r.Path
	if q := r.Query.Encode(); q != "" {
		path += "?" + q
	}
	body := write.BodyShown(r.Path, a.Order.ContentType, r.Body)
	if len(body) > maxAskBody {
		body = body[:maxAskBody] + "..."
	}
	if a.Unpreviewed {
		lines := []string{ui.Bridge("No preview: the Chancery makes this token even when asked only for a preview.", "Request: "+r.Method+" "+path)}
		if !ui.Naming() {
			lines = append(lines, "Change: "+storyChange(r))
		}
		if !ui.StoryNaming() && body != "" {
			lines = append(lines, "Body: "+body)
		}
		lines = append(lines, mintsText())
		return hud.StoneAsk{Title: ui.Bridge("CONFIRM: make a writ's token?", "CONFIRM: "+a.Command), Lines: lines, Confirm: true}
	}
	verdict := "Dry run (dryRun=All): " + a.Preview.Text()
	if a.Preview.OK() {
		verdict = "Dry run (dryRun=All): " + strings.TrimSpace(strings.TrimPrefix(a.Preview.Text(), "200 "))
	}
	lines := []string{ui.Bridge("The Chancery's preview answered first; nothing has changed yet.", "Request: "+r.Method+" "+path)}
	if !ui.Naming() {
		lines = append(lines, "Change: "+storyChange(r))
	}
	if !ui.StoryNaming() && body != "" {
		lines = append(lines, "Body: "+body)
	}
	lines = append(lines, ui.Bridge("Preview: "+storyVerdict(a.Preview), verdict))
	if !a.Preview.OK() {
		lines = append(lines, ui.Bridge("Nothing can be sent: the Chancery refused the preview.", "Nothing can be sent: the dry run failed."))
	}
	return hud.StoneAsk{Title: ui.Bridge("CONFIRM: send this change to the realm?", "CONFIRM: "+a.Command), Lines: lines, Confirm: a.Preview.OK()}
}

func tokenWarning(level write.Level) string {
	if level == write.Seamless {
		return ui.Bridge("Asking for a writ's token makes a real token that opens doors until it expires, and there is no preview: the Chancery makes it even when asked only for a preview. Y makes it now; no CONFIRM follows at this level.",
			"A ServiceAccount token request (kubectl create token) mints a real token, valid until it expires, and has no preview: the API server mints it even for a dry run. Y mints it now; at DANGEROUSLY SEAMLESS no CONFIRM follows.")
	}
	return ui.Bridge("Asking for a writ's token makes a real token that opens doors until it expires, and there is no preview: the Chancery makes it even when asked only for a preview. After this warning a CONFIRM asks, and its Y makes the token.",
		"A ServiceAccount token request (kubectl create token) mints a real token, valid until it expires, and has no preview: the API server mints it even for a dry run. After this warning a CONFIRM asks, and its Y mints the token.")
}

func mintsText() string {
	return ui.Bridge("Asking for a writ's token makes a real token at once: the Chancery makes it even for a preview, so there is none. Y makes it; it opens doors until it expires.",
		"A ServiceAccount token request (kubectl create token) has no preview: the API server mints a real token even for a dry run. CONFIRM mints it, and it is valid until it expires.")
}

func storyChange(r write.Request) string {
	var named struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
	}
	if r.Method == http.MethodPost {
		json.Unmarshal(r.Body, &named)
	}
	ref, sub := WrittenRef(r.Path, named.Metadata.Name)
	verb := map[string]string{http.MethodPost: "make", http.MethodPut: "rewrite", http.MethodPatch: "change", http.MethodDelete: "remove"}[r.Method]
	if r.Method == http.MethodPost && sub == "eviction" {
		verb, sub = "send away", ""
	}
	kind := storyKind(ref.Kind)
	if kind == ref.Kind {
		kind = ui.Name(ref.Kind)
	}
	object := kind + " " + ref.Name
	if ref.Namespace != "" {
		object = kind + " " + ref.Namespace + "/" + ref.Name
	}
	if ref.Name == "" {
		object = kind + "s"
		if ref.Namespace != "" {
			object += " in " + ref.Namespace
		}
	}
	if sub != "" {
		object += " (" + sub + ")"
	}
	return verb + " " + object
}

func openingLines(a live.StoneAsk) []string {
	o := a.Opens
	object := o.Name
	if o.Namespace != "" {
		object = o.Namespace + "/" + o.Name
	}
	kind := map[string][2]string{"Pod": {"hut", "pod"}, "Service": {"depot", "service"}, "Node": {"Keep", "node"}}[o.Kind]
	target := ui.Bridge(kind[0], kind[1]) + " " + object
	if o.Container != "" {
		target += ", " + ui.Bridge("worker ", "container ") + o.Container
	}
	var opens string
	switch o.Channel {
	case "exec":
		opens = ui.Bridge("a shell in ", "exec in ") + target
	case "attach":
		opens = "attach to " + target
	case "portforward":
		opens = ui.Bridge("a port-forward to ", "port-forward to ") + target
		if ports := forwardPorts(a.Command); len(ports) > 0 {
			opens += ", ports " + strings.Join(ports, " ")
		}
	case "proxy":
		opens = "the proxy of " + target + ", path " + o.Path
	}
	lines := []string{"Opens: " + opens}
	if a.Local != "" {
		if a.Writing {
			lines = append(lines, ui.Bridge("On this computer: copies into ", "On this computer: writes into ")+a.Local)
		} else {
			lines = append(lines, ui.Bridge("On this computer: copies from ", "On this computer: reads from ")+a.Local)
		}
	}
	if len(o.Command) > 0 {
		lines = append(lines, ui.Bridge("Runs: ", "Command: ")+strings.Join(o.Command, " "))
	}
	if o.Stdin || o.TTY {
		lines = append(lines, ui.Bridge("Every key typed goes to the worker until ctrl-p ctrl-q.", "Every key goes to the container (stdin, TTY) until ctrl-p ctrl-q."))
	}
	return append(lines, ui.Bridge("No preview: a connection has none. It opens on CONFIRM; the ledger notes it.",
		"No dry run: a connection has none. It opens on CONFIRM; the ledger records it."))
}

func forwardPorts(command string) []string {
	words, err := kubectltree.Words(command)
	if err != nil {
		return nil
	}
	at := slices.Index(words, "port-forward")
	if at < 0 {
		return nil
	}
	valued := map[string]bool{"-n": true, "--namespace": true, "--address": true, "--pod-running-timeout": true}
	var args []string
	for i := at + 1; i < len(words); i++ {
		w := words[i]
		if strings.HasPrefix(w, "-") {
			if valued[w] {
				i++
			}
			continue
		}
		args = append(args, w)
	}
	if len(args) < 2 {
		return nil
	}
	return args[1:]
}

func storyVerdict(a write.Answer) string {
	if a.OK() {
		return "the change would be accepted."
	}
	return "refused (" + a.Text() + ")."
}

func (g *Game) toggleStone() {
	if g.Sess == nil {
		g.Settings.Stone = hud.NextStoneMode(g.stoneKubectl())
	} else {
		g.Settings.Stone = hud.NextStoneMode(g.Sess.Hud.StoneKubectl())
		g.Sess.Hud.SetStoneMode(g.Settings.Stone)
	}
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}

var writtenKinds = map[string]string{"pods": "Pod", "deployments": "Deployment", "statefulsets": "StatefulSet", "daemonsets": "DaemonSet", "replicasets": "ReplicaSet",
	"nodes": "Node", "services": "Service", "jobs": "Job", "cronjobs": "CronJob", "horizontalpodautoscalers": "HorizontalPodAutoscaler", "configmaps": "ConfigMap",
	"secrets": "Secret", "namespaces": "Namespace", "persistentvolumeclaims": "PersistentVolumeClaim", "persistentvolumes": "PersistentVolume",
	"serviceaccounts": "ServiceAccount", "ingresses": "Ingress", "poddisruptionbudgets": "PodDisruptionBudget"}

func WrittenRef(path, name string) (write.Ref, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var ref write.Ref
	switch {
	case len(parts) >= 3 && parts[0] == "api":
		ref.GVR, parts = write.GVR{Version: parts[1]}, parts[2:]
	case len(parts) >= 4 && parts[0] == "apis":
		ref.GVR, parts = write.GVR{Group: parts[1], Version: parts[2]}, parts[3:]
	default:
		return write.Ref{Name: path}, ""
	}
	if len(parts) >= 3 && parts[0] == "namespaces" && parts[2] != "finalize" && parts[2] != "status" {
		ref.Namespace, parts = parts[1], parts[2:]
	}
	ref.Resource, ref.Name = parts[0], name
	if len(parts) > 1 {
		ref.Name = parts[1]
	}
	ref.Kind = writtenKinds[ref.Resource]
	if ref.Kind == "" {
		ref.Kind = ref.Resource
	}
	sub := ""
	if len(parts) > 2 {
		sub = strings.Join(parts[2:], "/")
	}
	return ref, sub
}

func stoneVerb(method string) string {
	switch method {
	case http.MethodPost:
		return ui.Bridge("made", "create")
	case http.MethodPut:
		return ui.Bridge("rewritten", "replace")
	case http.MethodPatch:
		return ui.Bridge("changed", "patch")
	case http.MethodDelete:
		return ui.Bridge("removed", "delete")
	}
	return method
}

func (s *Session) recordStoneWrite(order string, w kubectltree.Write) {
	wr := s.Writer
	if wr == nil {
		return
	}
	ref, sub := WrittenRef(w.Path, w.Name)
	object := ref.String()
	if sub != "" {
		object += " (" + sub + ")"
	}
	answer := "OK"
	if w.Code < 200 || w.Code > 299 {
		answer = w.Status
	}
	e := write.Entry{At: wr.now().UTC(), Context: wr.gate.Context().Name, Level: wr.level.String(), Order: order, Object: object, Code: w.Code, Answer: answer}
	if err := wr.ledger.Append(e); err != nil {
		s.Hud.Toast("The ledger file could not be written: "+err.Error(), true)
	}
	line := ledgerLine{at: e.At, verb: stoneVerb(w.Method), target: ref, code: w.Code, answer: answer}
	wr.entries = append(wr.entries, line)
	if len(wr.entries) > ledgerShown {
		wr.entries = wr.entries[len(wr.entries)-ledgerShown:]
	}
	s.Echoes = Chronicled(s.Echoes, "WRITE "+e.Level+" "+line.text(), s.echoClock())
	s.refreshLive()
}
