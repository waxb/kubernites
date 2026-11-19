# Contributing to Kubernites

Pull requests are welcome. This page says what belongs here, how to build and test, the one rule that is never bent, and how a pull request is signed and merged.

## What belongs here

This repository holds Kubernites, the cluster tool: its view of a real Kubernetes cluster, its gates, the Speaking Stone, and the art, drawing, sound and interface it shares with the full game.

Good contributions:
- bug fixes, each with a test that fails without the fix;
- the realm showing a Kubernetes object or state it does not show yet, true to what Kubernetes does (check it against the documentation on kubernetes.io and link it);
- reading more kinds of clusters well: managed clouds, large clusters, clusters where the user may see only some namespaces;
- building, packaging and running on more platforms;
- tests and documentation.

What does not belong here:
- anything that widens a gate, goes around one, or sends a change the user did not order (see "The gates");
- telemetry, or any network request to anything but the cluster the user chose;
- the full game's campaign, simulation and teaching, which are not open source and are not in this repository;
- a new module dependency without a reason, or one whose license does not fit Apache-2.0.

If you are unsure, open an issue first and ask.

## The gates

Kubernites reads by default and changes a cluster only at a level the user chose before connecting (README, "Read-only by default"). Every request to a cluster leaves through a gate: the read-only gate in `pkg/live/readonly.go` at READ ONLY, the write gate in `pkg/live/write` at WRITES and DANGEROUSLY SEAMLESS. Only the write gate builds a request other than a GET and the access questions, and the Stone's kubectl (`pkg/live/kubectltree`) sends through it. A pull request must not widen a gate, add a way around one, send a change the user did not order, or import a client that can write (`k8s.io/client-go/kubernetes`, `k8s.io/client-go/dynamic`, `k8s.io/kubectl`, `k8s.io/cli-runtime`) anywhere else. `TestTheLivePackageCanOnlyBuildReads`, `TestOnlyTheWriteGateBuildsARequestThatIsNotAGet` and `TestEveryConfigTheTreeHandsToKubectlGoesThroughTheGate` check part of this from the source, the `gates` CI job runs the gates' tests on every pull request, and review checks the rest: changes to `pkg/live`, `pkg/kubectl`, the Stone, the Scriptorium and the live screens need the owner's review (`.github/CODEOWNERS`).

## Building and testing

You need Go 1.26.1 or later. On Linux, running Kubernites and the client's tests needs X11 and OpenGL (see the README) and, without a display, `xvfb`.

```
gofmt -l .
go vet ./...
go build ./...
go test -short ./...
```

On Linux without a display, run the tests as `xvfb-run -a go test -short ./...`. The whole short run takes about three minutes on four cores and needs about 1.5 GB of memory. The gates' tests alone:

```
go test -count=1 -run '^(TestOnlyReadsAndAccessReviewsLeaveTheProcess|TestARequestOutsideTheBasePathIsRefused|TestAConnectionUpgradeIsRefusedEvenOnAGet|TestAnEncodedPathIsRefused|TestAWriteForcedThroughTheConnectionIsRefusedBeforeTheServer|TestTheLivePackageCanOnlyBuildReads|TestAFullLiveSessionSendsNoWrite|TestOnlyTheWriteGateBuildsARequestThatIsNotAGet|TestOnlyTheConfirmedChoiceBuildsTheGateAndOnlyCONFIRMIssuesATicket|TestTheGateIsNeverBuiltAtReadOnly|TestAtWritesAChangeNeedsItsTicketAndTheTicketIsUsedOnce|TestARevokedTicketPassesNothing|TestEachClassNeedsItsWarningEvenForAPreview|TestAStaleRealmSendsNoChangeAndNoPreview|TestNothingOutsideTheAPIOrOddPassesAtAnyLevel|TestAtWritesNothingButReviewsAndDryRunsLeaveUntilConfirm|TestTheLedgerKeepsOneLinePerWriteAndNoSecretValue|TestSecretValuesAreMaskedUntilRevealedKeyByKey|TestEveryConfigTheTreeHandsToKubectlGoesThroughTheGate|TestEveryRequestTheTreeMakesPassesTheGate|TestAtReadOnlyNoWriteLeavesTheGame|TestTheKubeconfigIsNeverWritten|TestTheTreeUsesOnlyTheChosenKubeconfigAndContext|TestPluginsAreOff)$' ./pkg/live/...
```

## Style

- `gofmt`, and no code comments: names and tests carry the meaning.
- No emojis, in code, text or commit messages.
- Captures render the same picture every run: nothing that reaches the screen may depend on Go's map order or the wall clock.
- Every thing in the realm has one story name, everywhere. `pkg/lexicon/lexicon.yaml` is the source; propose a new name in an issue first.

## Sign your commits

Every commit needs a `Signed-off-by` line with the name and email of its author:

```
git commit -s -m "fix: the harbor shows a Gateway with no address"
```

The sign-off says you agree to the Developer Certificate of Origin 1.1 (https://developercertificate.org/): you wrote the change, or otherwise have the right to submit it under this project's license. The `dco` CI job fails a pull request with a commit that has no sign-off from its author. To sign off commits already on your branch, run `git rebase --signoff main` and push again with `--force-with-lease`.

## The license of your contribution

Your contribution is licensed under the Apache License 2.0, like the rest of this repository (section 5 of `LICENSE`). The same code also ships inside the full Kubernites game, which is sold on itch.io and is not open source. Apache-2.0 allows that, and your sign-off confirms you can contribute under it.

## How a pull request is merged

Pull requests are squash merged: each one lands on `main` as one commit, with the pull request's title as its subject. Keep the title a short conventional message (`fix:`, `feat:`, `docs:`, `test:`).

Kubernites is developed in a private repository that also holds the full game. After a merge, the maintainer brings your commit into that repository unchanged (author, date and message, with one line naming the commit here), and later syncs come back to this repository as commits named `sync from private <commit>`. A sync never undoes a merged contribution.
