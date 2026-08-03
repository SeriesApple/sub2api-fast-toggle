# Third-Party Notices

This repository contains original proxy and toggle code. It interoperates with
the following separately maintained software and services.

## Sub2API

- Project: <https://github.com/Wei-Shaw/sub2api>
- License reported by upstream on 2026-08-03: LGPL-3.0
- License file: <https://github.com/Wei-Shaw/sub2api/blob/main/LICENSE>

Sub2API is not redistributed in this repository or in the final proxy image.
Users install and operate Sub2API separately.

## Go

- Project: <https://go.dev/>
- Source: <https://github.com/golang/go>
- License: BSD-style license

The statically compiled proxy includes Go runtime and standard-library code.
A copy of the Go license is included in the repository and final image as
[LICENSES/Go-BSD-3-Clause.txt](LICENSES/Go-BSD-3-Clause.txt).

## Container Build Inputs

The Dockerfile uses the official `golang:1.26.4-bookworm` image as a build stage
and copies Debian's `ca-certificates` bundle into the final scratch image. The
package copyright notice is copied to
`/licenses/ca-certificates-copyright.txt` in that image. These build inputs
remain subject to their own upstream license and notice files.
