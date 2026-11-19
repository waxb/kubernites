## What this changes

## How you checked it

## Does this change what Kubernites can send to a cluster?

- [ ] No. It adds, removes and changes no request to a cluster, and does not touch `pkg/live/readonly.go`, `pkg/live/write`, `pkg/live/kubectltree` or their tests.
- [ ] Yes. Say which requests, why, at which levels, and how the gates still hold: nothing but reads at READ ONLY, and no change the user did not order (README, "Read-only by default").

## Checklist

- [ ] Every commit is signed off (`git commit -s`), as CONTRIBUTING.md says.
- [ ] `gofmt -l .` prints nothing, and `go vet ./...` and `go test -short ./...` pass.
- [ ] A fix comes with a test that fails without it.
