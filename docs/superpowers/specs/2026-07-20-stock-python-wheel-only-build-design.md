# Stock Python Wheel-Only Build Design

## Goal

Make the VibeDeploy aggregate route for `stock_workforce-compose` reliably build and start the stock history backtest project on the current Apple Silicon host.

## Root Cause

The first aggregate child, `stock-python-back`, uses a database-backed `Dockerfile.project` that installs `build-essential` and `gcc` from Debian before installing Python requirements. Two real deployments failed when Debian CDN requests routed through Docker Desktop's proxy returned HTTP 502. The failed packages later returned HTTP 200, confirming an intermittent proxy/CDN path rather than a port or container conflict.

The compiler packages are not required by the current dependency graph. A preflight in `python:3.11-slim` on the shared Docker network successfully downloaded and resolved every direct and transitive requirement with `--only-binary=:all:` for Python 3.11 ARM64.

## Persistent Change

Update VibeDeploy route script ID 541, the persistent source for `stock-python-back`'s `Dockerfile.project`:

- Keep `python:3.11-slim`, `/app`, and the existing requirements path.
- Remove the Alpine compatibility probe and Debian APT/GCC installation layer.
- Install requirements with binary wheels only.
- Add bounded pip retries and a longer network timeout.
- Keep the Tsinghua Python package index already used by the project.

The intended dependency layer is:

```dockerfile
COPY stock_view/stock_python_back/requirements.txt .
RUN python -m pip install \
    --only-binary=:all: \
    --retries 8 \
    --timeout 60 \
    --no-cache-dir \
    -r requirements.txt \
    -i https://pypi.tuna.tsinghua.edu.cn/simple
```

The database update must use an optimistic content-hash guard so an unexpected concurrent edit is not overwritten. The exact previous template is retained for rollback.

## Failure Behavior

Binary-only installation deliberately rejects a future dependency that has no compatible wheel. This produces an immediate and actionable Python dependency error instead of silently requiring a compiler and reintroducing the unstable Debian package download.

Pip retries cover short-lived Python index or proxy failures without creating unbounded deployment time.

## Verification

Verification proceeds in increasing scope:

1. Validate the proposed Dockerfile contract: no `apt-get`, `apk`, `gcc`, or `build-essential`; binary-only pip installation is required.
2. Build the stock Python project dependency image from the real repository context.
3. Trigger VibeDeploy project 88, route 159, through its real aggregate deployment API.
4. Confirm every expected stock business container is running, uses restart policy `no`, and is attached only to `vibedeploy-shared`.
5. Inspect container health/logs and exercise the exposed stock application endpoints.

## Rollback

If the wheel-only image build fails because of this change, restore script ID 541 to its exact pre-change content using the saved hash guard, rematerialize the route, and report the failing dependency. A failure in an unrelated child project is diagnosed separately and does not justify silently reverting a verified Python build improvement.
