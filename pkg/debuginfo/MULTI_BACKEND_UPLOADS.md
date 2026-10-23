# Multi-backend debuginfo uploads: protocol sketch

Let a collector aggregate upload negotiation while clients PUT the artifact to
each backend's signed URL. The collector stores no artifact bytes. Clients call
the collector to finalize each successful upload independently.

This sketch adds protocol fields and exercises the new response shape against
Parca's single-backend store. It does not implement the collector aggregator or
an agent upload scheduler. This repository's `GrpcUploadClient` transfers one
artifact; the full client negotiation and signed-URL loop lives outside it.

## Additive wire changes

- `InitiateUploadRequest.supports_multiple_uploads`: capability opt-in.
- `InitiateUploadResponse.upload_targets`: repeated `UploadInstructions`.
- `InitiateUploadResponse.retry_required`: some destinations remain unresolved.
- `ShouldInitiateUploadResponse.retry_required`: a negative answer is temporary,
  for example because another upload is in progress.

Keep the singular `upload_instructions` field at its existing field number.
An upgraded Parca returns a one-element list to opted-in clients and the old
singular field to legacy clients. Never populate both. A new client accepts
either shape because older servers ignore the capability bit.

A fanout collector rejects initiation without the capability bit with
`FailedPrecondition`, before starting any downstream uploads. Do not return the
first target in the legacy field and silently lose the rest. A transparent
single-backend forwarder can keep its existing behavior.

## Reuse upload_id for completion routing

`upload_id` is already an opaque value echoed in `MarkUploadFinished`. No new
backend identifier or completion RPC is necessary. The collector replaces each
downstream upload ID with its own token and returns the signed URL unchanged.

The token resolves to:

```text
authenticated tenant
artifact build ID and type
configured destination identity
downstream upload ID
expiration
```

Use an unguessable ID backed by shared session storage, or an authenticated,
encrypted token. The implementation choice is open. Either must work across
collector replicas and bind the route to the authenticated tenant and artifact.
Do not encode an arbitrary client-supplied endpoint to dial. Do not log tokens
or signed URLs.

On completion, validate the token, restore the downstream upload ID, and call
exactly that destination. Never broadcast completion. A nested collector wraps
the downstream token and restores it when forwarding completion.

Keep destination identities stable for the lifetime of outstanding uploads.
Removing a destination must produce an explicit error for its pending tokens,
not route them to a new backend that happens to occupy the same list position.

## Collector behavior

`ShouldInitiateUpload` queries all configured destinations selected for the
artifact and ORs their `should_initiate_upload` answers. An affirmative answer
can proceed despite another backend being unreachable. Carry unresolved checks
in `retry_required`. If no backend says yes and any check failed, return a
retryable RPC error rather than a definitive no. Preserve retry hints from
downstream responses.

`InitiateUpload` checks destinations again; do not remember only the backends
that said yes during the client's earlier preflight. For each destination:

1. A definitive negative preflight needs no instruction.
2. A temporary negative or failed check leaves negotiation incomplete.
3. An affirmative answer triggers downstream initiation with the capability bit.
4. Flatten successful downstream lists, also accepting legacy singular replies.
5. Wrap each upload ID and propagate downstream `retry_required`.

Return successful instructions even when other destinations fail, with
`retry_required = true`. If no usable instructions exist and a backend failed,
return an RPC error. A permission or configuration error needs operator action;
clients must bound retries rather than loop forever. If all destinations
definitively decline, an empty list with `retry_required = false` is valid.

A backend can change state between preflight and initiation. Recheck a refused
initiation rather than interpreting every `FailedPrecondition` as "present".
In current Parca, `AlreadyExists` can also mean the same invalid artifact hash
was rejected. That is a terminal refusal of these bytes, not proof of usable
debuginfo. Keep its diagnostic reason when reporting failures.

For older Parca backends, the existing `ReasonUploadInProgress` response needs
translation to a retry hint. The new field avoids depending on that reason
string after downstreams upgrade. Do not assume any old negative response
means the backend has finished storing the artifact.

## Client loop

The following is pseudocode, not an implemented scheduler:

```text
check = ShouldInitiateUpload(artifact)
if not check.should_initiate_upload:
    retry later if check.retry_required; otherwise stop

plan = InitiateUpload(artifact, supports_multiple_uploads=true)
targets = plan.upload_targets, or [plan.upload_instructions] for old servers

for each target independently:
    open the artifact from its beginning
    PUT to target.signed_url
    if transfer succeeds:
        MarkUploadFinished(target.build_id, target.upload_id, target.type)

retain failed transfers and unacknowledged completions for bounded retries
retry completion without re-uploading when only completion failed
after resolving returned targets, renegotiate if plan.retry_required
release the artifact only after every target and unresolved destination is resolved
```

Do not abandon B because A's PUT or completion failed. Successful completion of
A does not mean B is complete. Repeating completion for the same upload must be
safe. A stale token must not finalize a newer upload for the same build ID.

Avoid re-initiating known pending uploads; retain their instructions while they
remain usable. Signed URLs can expire, so transfer errors may require renewed
negotiation rather than replaying the same PUT indefinitely.

## Boundaries and unresolved work

- The initial aggregator supports signed-URL destinations only. Existing GRPC
  instructions mean "call Upload on the service that issued this response".
  Returning them unchanged from a collector would require byte proxying. Reject
  unsupported destinations explicitly; adding a direct gRPC endpoint and its
  authentication contract is separate work.
- Lost initiation responses can leave a downstream upload in progress until it
  expires. The retry hint prevents treating this as completed delivery, but does
  not provide immediate recovery. A request idempotency key or a session lookup
  can address this later.
- Do not use `force=true` to recover unknown state. It can invalidate a live
  upload. Retrying a partially successful forced fanout also needs a session
  that remembers completed destinations; initially reject forced fanout rather
  than claiming the normal retry loop handles it.
- There is no atomic multi-backend commit, collector-owned delivery guarantee,
  or promise of retries after the client exits. The client retains the artifact
  and reports an unresolved delivery when its retry budget expires.
- Production collector work still needs bounded concurrent calls, deadlines,
  token lifecycle/authentication, destination configuration, and tests for mixed
  outcomes, nested collectors, lost responses, and completion routing.

## Rollout

Deploy the additive server changes, then update agents to negotiate the list
form, execute all instructions, and honor retry hints. Enable fanout only for
capable clients and signed-URL destinations. Old agents can keep talking to a
single Parca backend without changing their request or response shape.
