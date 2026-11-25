# Changelog

Every version of Kubernites, newest first. A release goes out only when the version and this file both moved (`scripts/check-release.sh`).

## 0.15.2 (2026-11-25)

- Kubernites no longer quits about a second after it starts on some Macs, as it did on macOS 27.2.1 (issue #1). It moves to Ebitengine 2.10.6, whose window code no longer calls itself without end when something subclasses the window's view while Kubernites runs (hajimehoshi/ebiten#3740). This repository's own build had the same fault.
- A crash, or an error that ends Kubernites, is also written to `crash.log`, under a line naming the version that ran: beside the save for the game from itch.io (`~/.kubernites/crash.log`), in the settings folder for this repository's build (`~/Library/Application Support/kubernites` on macOS, `~/.config/kubernites` on Linux, `%AppData%\kubernites` on Windows). A window opened from the Finder or the Explorer has no terminal, so before this a crash left nothing to report.
- The macOS app from itch.io asks for macOS 15 or later, and its program now asks for the same: the 0.15.1 program asked for macOS 26 while the app said 12.

## 0.15.1 (2026-11-22)

- In LIVE CLUSTER the top bar shows the cluster's CPU, memory and Pods requested of allocatable, counted as the scheduler counts requests, with the campaign's food and housing in story mode. A number the cluster does not give is left out.
- A node's card in LIVE CLUSTER shows its CPU, memory and Pods as bars, requested of allocatable, with metrics-server's usage when it answers, its instance type and zone, and what the control plane's own Pods request on a control-plane node. It showed no bar before.
- The Dashboard opens in LIVE CLUSTER (C, or DASHBOARD beside EVENTS and KUBE): the nodes as the scheduler sees them, Pending Pods with the scheduler's reason, workloads below their desired replicas, Services with no ready endpoint, the busiest Pods and each node's ephemeral storage, under four gauges that count Ready nodes, Ready Pods, workloads at their replicas and Services with a ready endpoint. It only reads.
- LIVE CLUSTER draws a big cluster faster: a frame of 30 nodes and 580 Pods takes about a third less CPU at the closest zoom and a quarter less zoomed out, and makes less than half the garbage, because buildings off the screen are no longer drawn and the town's layout is no longer worked out again on every frame. Nothing on the screen looks or answers differently.
- The campaign's Keep card shows its housing bar beside its food bar, and its size line fits on one row with Kubernetes names.
- In the Handbook, a page's card keeps the pointer while it is over the card, so LEARN, ON A CREW and ALMANAC can be clicked wherever the card is drawn, also to the left of its plate and over other plates. The card no longer jumps to the page of a plate beneath it, a card opened by pointing at a plate stays open on the way to its buttons, and a card is never drawn over its own plate.
- The gold frame the guide puts on a Handbook plate is no longer drawn over an open card: it stops at the card's edge, and at the edge of the map for a plate scrolled half out of sight.
- When the realm pauses itself, the tips of the top bar's buttons keep off the PAUSED box.
- The notices at the top right stay in sight while the short Speaking Stone is open, as the step's boxes do, so a notice being read no longer vanishes while a command is typed.

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
