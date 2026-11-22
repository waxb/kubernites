# Security

Kubernites connects to real clusters with real credentials. It promises to only read unless the user chose a write level for that context before connecting, and even then to send no change the user did not order. A way to break that promise is a security problem.

## Reporting a problem

Report it privately, through GitHub's private vulnerability reporting: on https://github.com/waxb/kubernites, open the Security tab and click Report a vulnerability (or go straight to https://github.com/waxb/kubernites/security/advisories/new). Please do not open a public issue or pull request about it before a fix is released.

## What counts

These are security problems:
- anything that sends a request at READ ONLY other than the reads the read-only gate allows (README, "Read-only by default");
- anything that sends a change the user did not order: a change at WRITES without its preview and CONFIRM (apart from a card order whose dry run an admission webhook cannot take, sent after its own warning, and a ServiceAccount token request, which the API server carries out even as a dry run and so is sent with no preview, after its own warning and CONFIRM), a change while the picture is stale, or at any level a change sent by time passing, by trouble shown in the realm, by the MENU, by a plain key, or by anything the cluster holds;
- anything that lets a request skip its class warning (Secrets, reaching into a workload, destroying more than one thing, a change without a preview, minting a token), shows a Secret's value before the user reveals it, or writes one to the ledger;
- anything that lets the Stone's kubectl write the kubeconfig, reach another kubeconfig, context or server, run a plugin, an outside editor, or any outside program other than the kubeconfig's own credential plugin, or fetch from the web;
- anything that reaches into a workload (`exec`, `attach`, `port-forward`, `cp`, `debug`, `proxy`) at READ ONLY, without its warning, or at WRITES without its CONFIRM; a port-forward that listens on any address but 127.0.0.1 or stays open after its connection closes; a `cp` that reads or writes a local path outside the one typed (a relative path is taken from `~/Downloads/Kubernites`), writes `~/.kube` or the kubeconfig chosen at connect, or replaces an existing local file without asking; a key typed into the Stone's terminal that reaches anything but that container, or ctrl-p ctrl-q reaching the container;
- anything that sends a kubeconfig's credentials, or what Kubernites read from a cluster, anywhere but that cluster's API server;
- a way for what a cluster holds (names, labels, annotations, events, logs) to make Kubernites run code or send a request.

A wrong drawing, wrong information or a crash is an ordinary bug: open an issue.

## Supported versions

The latest release and `main`.
