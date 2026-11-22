# Changelog

Every version of Kubernites, newest first. A release goes out only when the version and this file both moved (`scripts/check-release.sh`).

## 0.15.0 (2026-11-22)

- The first release of this repository.
- A flag that ends its command, such as `--now).` in the KILL tip, stays on the line with its command instead of taking the next word with it to a new line.

## 0.14.10 (2026-11-21)

- When a panel leaves no room above a building, its name goes below the building instead of across it, so it no longer covers the Citadel's flag or a tower's roof. The name also stays clear of the arrow that points at the next step's building.
- The hint to enlarge a small window keeps clear of the chapter map's stops and of the title menu's border.
- A line in an order scroll picks the most even of its clause-end breaks, and starts a new sentence on its own line when that adds no line.
- A command in a tip keeps a flag with its value, so `-n` and the namespace stay on one line.
- A card whose name is too long keeps its whole kind: CUSTOM RESOURCE KIND (FOREIGN HALL in story) for a hall of custom resources, not just its first word.
- The picker's WRITES box says Kubernites draws the cluster as it does at READ ONLY.

## 0.14.9 (2026-11-21)

- A selected building's name, and the plate of the Citadel across the sea, sit above the building at every zoom, and above a Keep's hands meter.
- With Kubernetes names, what the cluster answers is shown word for word. In story mode Kubernites no longer turns the cluster's words into story words; it says what was not done and that NAMES shows why, and a refused order says what these credentials may not do.
- A server name with no DNS record says what to check already when Kubernites asks who you are, not only when it opens the cluster.
- A vault is named for the claim it holds: PersistentVolumeClaim with Kubernetes names, Deed in story. It was named PersistentVolume, which the claim is not.
- A line in an order scroll breaks at the end of a clause when it can, the level box in the picker no longer touches the SEEN AS line, and the hint to enlarge a small window stays clear of the title buttons.

## 0.14.8 (2026-11-20)

Fixes from a fifth round of security review, fourteen findings, none of them high or medium:

- At WRITES the API groups are checked again before every change, the built-in ones included, so a version of a built-in group handed to an aggregated API server after connecting gets no dry run.
- A server address written without a scheme counts its user name and password in the remembered choice.
- The ledger records a change whose answer never arrives, a dry run the gate did not trust to change nothing, and a Secret change on a server whose address has a path, with its values masked; a `run` keeps no word after the Pod's name but its flags.
- A Secret made on the client side through `run` or `expose --overrides` is never printed through a template, and client-side output too long to mask is not shown.
- Masking held errors takes one pass however many values are known, and STOP ends it.
- On layouts without Latin letters, such as Greek and Hebrew, ctrl-p ctrl-q leaves the terminal.
- When a container ends its session, what is typed next is dropped until Enter gives the Stone's prompt back.
- The picker asks its warnings again when a context's API server or user entry changes, and a choices file written before the token warning had its own class no longer passes the no-preview warning.
- The WRITES warning names a change with no preview among the warnings asked once.

## 0.14.7 (2026-11-20)

Fixes from a fourth round of security review, twenty findings in all:

- A ServiceAccount token request has a warning of its own, minting a token, and one that asks for a dry run is refused at every level: the API server mints the token even then.
- At WRITES the API groups are checked again before each change to a group outside the built-in ones, so a group handed to an aggregated API server after connecting gets no dry run.
- A server address never shows its user name or password, whatever characters they hold, and a choices file that kept them is rewritten without them.
- The Stone checks a local path the way it will be opened (links, `..`, globs), walks a `~/.kube` reached through a link, and masks a Secret made on the client side in every output format.
- The ledger keeps the program a container session runs, never its arguments, and records a drain's cordon and how the drain ended.
- Control chords, and the Y that answers a warning or a CONFIRM, follow the letters the keyboard layout puts on the keys.
- The Stone's terminal drops a container's titles, hyperlinks and other string sequences.
- The picker names turning off the server's certificate check as such.

## 0.14.6 (2026-11-20)

- LIVE CLUSTER draws a cluster live: each kind is listed once, then watches stream every change; custom resources, new Services, Ingresses and Gateways appear as they come, and a watch the credentials forbid is shown as such.
- Changing a cluster sits behind a level chosen before connecting: READ ONLY by default, WRITES (every change previewed with a server dry run, then confirmed) and DANGEROUSLY SEAMLESS, with a warning before the first change of each kind: Secrets, reaching into a workload, destroying more than one thing, a change without a preview.
- Orders on the cards: scale, restart, roll back, pause and resume, set image or resources, change an HPA's range, delete and kill, trigger a CronJob, cordon, uncordon and drain a node.
- The Speaking Stone runs `kubectl` v1.37 itself, in the process, behind the same gate; the Scriptorium edits any object as YAML and applies it server-side.
- Reaching into workloads: a terminal in the Stone for `exec`, `attach`, `debug` and `run`, `port-forward` on 127.0.0.1 only, `cp`, and logs that follow.
- A ledger of every change and every session, kept beside the settings.
- Fixes from three rounds of security review of every way a request leaves Kubernites and every way a value reaches the screen or the disk.
- The title page has SETTINGS: sound, volumes, effects, fullscreen, edge scroll, NAMES, STONE and HINTS.
- Every card's and tab's command names what the card shows: a vault is its PersistentVolumeClaim (`pvc`, in its namespace), a hall of custom resources is its kind across its namespace (`ciliumendpoints.cilium.io`), and the API server answers with `cluster-info` and `version`.
- A live cluster opens at the closest zoom, where the art shows best.
- The title page and the context picker speak the live NAMES setting.
- A server whose name has no DNS record says what to check: a stopped, deleted or recreated cluster, or one reachable only inside its own network or VPN.
- `--version` and the title name the commit a build was made from.
