# Module: golang.org/x/crypto

## Overview

**golang.org/x/crypto** is the Go team's supplementary cryptography library — packages that live outside the standard library's `crypto/*` tree. It provides production-grade implementations of cryptographic algorithms including SSH, NaCl (sealed boxes), Argon2, bcrypt, HKDF, Curve25519, and others.

**Key Characteristics:**
- Maintained by the Go core team (golang.org/x namespace)
- Packages that are too specialized or in-progress for the standard library
- Widely relied upon across the Go ecosystem
- BSD-licensed, same license as the Go standard library

**Maintainers:** Go team (golang.org)

## Version Used

Current version in `go.mod`: **v0.53.0** (latest published tag as of 2026-07-07)

## Usage in gh-aw

### Files Using This Module

- `pkg/cli/secret_set_command.go` — `gh aw secrets set` command (production code)
- `pkg/cli/secret_set_command_test.go` — test coverage

### Sub-package Used

Only **one** sub-package is imported: `golang.org/x/crypto/nacl/box`

### Key APIs Utilized

#### Production
- `box.SealAnonymous` — encrypts a secret value using the anonymous sealed-box construction

#### Tests
- `box.GenerateKey` — generates ephemeral key pairs for round-trip testing
- `box.OpenAnonymous` — decrypts sealed boxes to verify correctness
- `box.AnonymousOverhead` — constant used to validate minimum ciphertext length

### Core Operation

```go
ciphertext, err := box.SealAnonymous(nil, []byte(plaintext), pk, rand.Reader)
```

This implements GitHub's required sealed-box construction (**Curve25519 + XSalsa20 + Poly1305**) for the Actions Secrets API. The 32-byte public key is fetched from `.../actions/secrets/public-key`, base64-decoded, length-validated, then converted to `*[32]byte` via the Go 1.17+ slice-to-array-pointer idiom.

## Research Summary

**Repository:** https://github.com/golang/crypto

**Latest Version:** v0.53.0 (verified against upstream tags — current)

### Recent Upstream Changes

Recent work is concentrated in packages gh-aw does **not** use:

- **`ssh`**: Hardened `ParseAuthorizedKey`/`ParseKnownHosts` to verify the declared key type matches the decoded key (matches OpenSSH behavior; fixes silent option-dropping).
- **`openpgp`**: Deprecation message made explicit — the package is now documented as *unmaintained, unsafe at any speed, and not receiving security fixes*, pointing users to the ProtonMail fork.

The `nacl/box` surface itself is intentionally tiny and has seen no functional change — it is stable and battle-tested.

### Security Posture

- `nacl/box` implements NaCl's sealed-box construction, a well-audited, widely-deployed primitive.
- The Curve25519 + XSalsa20 + Poly1305 combination provides authenticated encryption; ciphertext forgery is computationally infeasible.
- GitHub's Actions Secrets API mandates this specific construction; there is no alternative correct implementation.
- `golang.org/x/vuln` in CI guards against vulnerabilities in transitive crypto packages.

## Best Practice Alignment

gh-aw already follows every relevant best practice for `nacl/box` usage:

| Practice | Status |
|---|---|
| CSPRNG entropy via `crypto/rand.Reader` | ✅ |
| Public-key length validated before `*[32]byte` conversion | ✅ |
| Does **not** import deprecated `openpgp` package | ✅ |
| Test suite verifies a real `SealAnonymous`→`OpenAnonymous` round-trip | ✅ |
| Uses exported `box.AnonymousOverhead` constant for length assertions | ✅ |

## Improvement Opportunities

### Quick Wins

**None material.** The dependency is already at the latest tag and the usage is idiomatic.

### Feature Opportunities

No new `nacl/box` APIs to adopt — the sealed-box API is minimal and stable by design. GitHub's secrets API mandates the anonymous sealed-box form, so the current construction is the only correct choice.

### Optional Defense-in-Depth

Operating on a `[]byte` secret and zeroing it after encryption would be a defense-in-depth measure. In practice, Go's immutable strings and GC make this largely symbolic, so it is **not a defect** — just noted for completeness.

### General Notes

- The single-sub-package import keeps the trusted crypto surface small — a desirable state; no change recommended.
- Continue letting Dependabot/CI bump `x/crypto` promptly so security fixes in the broader module land (already current at v0.53.0).

## Recommendations

1. **Keep as-is.** This is an exemplary, minimal, correctly-tested use of `nacl/box`. No action required.
2. Continue letting Dependabot/CI bump `x/crypto` promptly so security fixes in the broader module land.

## References

### Documentation

- **Package Documentation:** https://pkg.go.dev/golang.org/x/crypto/nacl/box
- **Repository:** https://github.com/golang/crypto
- **Go Team Supplemental Packages:** https://pkg.go.dev/golang.org/x

### Related gh-aw Code

- **Production Usage:** `pkg/cli/secret_set_command.go`
- **Test Coverage:** `pkg/cli/secret_set_command_test.go`

### External References

- **GitHub Secrets API:** https://docs.github.com/en/rest/actions/secrets
- **NaCl Sealed Box:** https://libsodium.gitbook.io/doc/public-key_cryptography/sealed_boxes

---

**Last Reviewed:** 2026-07-07
**Reviewer:** GitHub Copilot (automated via issue #44248)
**Module Version:** v0.53.0

---

*This summary was generated based on Go Fan analysis methodology. For the latest information, always check the upstream repository.*
