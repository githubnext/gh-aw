---
title: Bash Command Parser Specification
description: W3C-style specification for the copilot SDK driver bash command parser and conformance test generation
sidebar:
  order: 1370
---

# Bash Command Parser Specification

**Version**: 1.0.0  
**Status**: Draft Specification  
**Latest Version**: [bash-command-parser-specification](/gh-aw/specs/bash-command-parser-specification/)  
**Editor**: GitHub Agentic Workflows Team

---

## Abstract

This specification defines the behavior of the bash command parser used by `actions/setup/js/copilot_sdk_driver.cjs` via `actions/setup/js/bash_command_parser.cjs`. It formalizes segment splitting, command-name extraction, deduplicated pipeline extraction, and conformance testing. It also defines a language-agnostic method for generating and verifying parser test suites.

## Status of This Document

This document is a draft extracted from the implementation and tests in:

- `actions/setup/js/bash_command_parser.cjs`
- `actions/setup/js/bash_command_parser.test.cjs`
- `actions/setup/js/fuzz_bash_command_parser_harness.cjs`
- `actions/setup/js/fuzz_bash_command_parser_harness.test.cjs`
- `actions/setup/js/copilot_sdk_driver.cjs`

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Formal Grammar](#3-formal-grammar)
4. [Parser Semantics](#4-parser-semantics)
5. [Driver Integration Semantics](#5-driver-integration-semantics)
6. [Conformance Test Suite Construction](#6-conformance-test-suite-construction)
7. [Model-Based Test Generation](#7-model-based-test-generation)
8. [Verification-Based Test Generation](#8-verification-based-test-generation)
9. [Machine-Readable Test Vectors](#9-machine-readable-test-vectors)
10. [Security Considerations](#10-security-considerations)
11. [References](#11-references)

---

## 1. Introduction

### 1.1 Purpose

The parser is a lightweight recognizer for shell command identifiers in chained/piped command text. Its output is used by the Copilot SDK permission handler to decide whether shell requests are allowed.

### 1.2 Scope

This specification defines:

- splitting on `&&`, `||`, `|`, `;`
- quote/subshell shielding during splitting
- executable token extraction from a segment
- deduplicated name extraction from pipeline text

This specification does **not** define a full POSIX shell parser.

---

## 2. Conformance

### 2.1 Requirements Notation

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### 2.2 Conformance Classes

- **Class S (Splitter)**: Implements segment splitting in §4.1.
- **Class E (Extractor)**: Implements single-segment extraction in §4.2.
- **Class P (Pipeline Extractor)**: Implements pipeline extraction in §4.3.
- **Class I (Integration Consumer)**: Implements driver behavior in §5.

A conforming implementation MUST satisfy all applicable MUST-level requirements for its class.

---

## 3. Formal Grammar

The grammar below is recognition-oriented and intentionally limited to parser behavior.

### 3.1 Splitting Grammar (EBNF)

```ebnf
command_text   = { unit } ;
unit           = single_quoted | double_quoted | subshell | operator | other ;
operator       = "&&" | "||" | "|" | ";" ;

single_quoted  = "'" , { ? any char except "'" ? } , [ "'" ] ;
double_quoted  = '"' , { dq_char | escape } , [ '"' ] ;
escape         = "\" , ? any char ? ;
dq_char        = ? any char except unescaped '"' ? ;

subshell       = "$(" , subshell_body ;
subshell_body  = { subshell_char | nested } ;
nested         = "(" , subshell_body , ")" ;
subshell_char  = ? any char not terminating current depth ? ;

other          = ? any other single char ? ;
```

### 3.2 Segment Extraction Grammar (EBNF)

```ebnf
segment        = ws , { env_assign , ws } , core ;
env_assign     = ident , "=" , nonspace* ;
ident          = ("_" | letter) , { "_" | letter | digit } ;

core           = negation | brace | keyword | redirection | word | empty ;
negation       = "!" , ws , core ;
brace          = ("{" | "}") , ws , core ;
keyword        = "then" | "else" | "elif" | "fi" | "do" | "done"
               | "esac" | "in" | "function" | "time" | "coproc" ;
redirection    = ("<" | ">") , nonspace*
               | digits , ("<" | ">" | "&") , nonspace* ;
word           = nonspace , { nonspace } ;
```

---

## 4. Parser Semantics

### 4.1 `splitOnPipelineOperators(commandText)`

1. Non-string or empty/falsy input MUST return `[]`.
2. The implementation MUST split at top-level operators `&&`, `||`, `|`, and `;`.
3. Operators inside single quotes, double quotes, or `$(` `)` regions MUST NOT split.
4. Output segments MUST be trimmed.
5. Empty segments MUST be removed.
6. The function SHOULD be non-throwing for malformed input.

### 4.2 `extractCommandName(segment)`

1. Non-string or blank segment MUST return `null`.
2. Leading environment assignments (`IDENTIFIER=\S*`) MUST be stripped repeatedly.
3. If the first token is redirection (`^[<>]` or `^\d+[<>&]`), return `null`.
4. If the first token is `!`, `{`, or `}`, extraction MUST recurse on the remainder.
5. If the first token is a shell keyword (`then`, `else`, `elif`, `fi`, `do`, `done`, `esac`, `in`, `function`, `time`, `coproc`), return `null`.
6. Otherwise return the first token.

### 4.3 `extractCommandNamesFromPipeline(commandText)`

1. Non-string or blank input MUST return `[]`.
2. Input MUST be split using §4.1.
3. Each segment MUST be extracted using §4.2.
4. Null extraction results MUST be ignored.
5. Returned command names MUST be deduplicated while preserving first-occurrence order.

---

## 5. Driver Integration Semantics

The parser output is consumed by `copilot_sdk_driver.cjs` in fallback shell-permission logic:

1. If multiple names are extracted (`length > 1`), **all** names MUST satisfy shell identifier rules.
2. If one name is extracted (`length === 1`), normal single-command matching applies, including exact full-command matching for literal rules that contain spaces.
3. If no names are extracted (`length === 0`), only exact full-command matching for literal-with-spaces rules is attempted; otherwise deny.
4. This preserves default-deny behavior when parsing cannot confidently identify commands.

---

## 6. Conformance Test Suite Construction

A language-independent test suite MUST contain:

- **Vector tests** for each parser function (split/extract/pipeline).
- **Robustness tests** for malformed/unbalanced inputs.
- **Deduplication/order tests** for repeated command names.
- **Integration tests** for fallback behavior in §5.

Implementations SHOULD consume machine-readable vectors and run identical assertions in each target language.

---

## 7. Model-Based Test Generation

Model-based tests MUST be generated from a finite-state splitter model with states:

- `TopLevel`
- `InSingleQuote`
- `InDoubleQuote`
- `InSubshell(depth>=1)`

Generation procedure:

1. Define token alphabet: command words, operators, quotes, `$(`, `)`, escapes, whitespace.
2. Build transition traces across all states and transitions.
3. Emit expected split points only in `TopLevel`.
4. Derive expected command extraction outputs from segment-first-token rules in §4.2.
5. Serialize generated vectors for cross-language execution.

---

## 8. Verification-Based Test Generation

Verification MUST include metamorphic/property-derived vectors:

1. **Whitespace invariance**: surrounding/inter-operator whitespace does not change extracted command names.
2. **Quoted operator shielding**: moving operators into quotes preserves single-segment behavior.
3. **Env-prefix invariance**: prepending env assignments does not change extracted command identifier.
4. **Redirection-suffix invariance**: appending redirection suffixes does not change extracted identifier.
5. **Duplicate-collapse invariance**: repeating same command across stages still yields one unique name.
6. **No-throw robustness**: malformed inputs SHOULD not throw and SHOULD keep return-shape guarantees.

---

## 9. Machine-Readable Test Vectors

Conforming projects SHOULD publish vectors in JSON with stable IDs and source tags:

- `source = "model-based"` for state-model-derived vectors
- `source = "verification"` for metamorphic/property-derived vectors

Reference vectors are provided in:

- `actions/setup/js/bash_command_parser_spec_vectors.json`

These vectors are executable in JavaScript via:

- `actions/setup/js/bash_command_parser_spec_vectors.test.cjs`

---

## 10. Security Considerations

This parser is not a shell sandbox and MUST NOT be treated as proof of command safety. Consumers MUST keep permission checks default-deny when command identification fails. Ambiguous/unparseable input SHOULD result in deny behavior at integration layer.

---

## 11. References

### 11.1 Normative

- RFC 2119: Key words for use in RFCs to Indicate Requirement Levels

### 11.2 Informative

- `/home/runner/work/gh-aw/gh-aw/actions/setup/js/bash_command_parser.cjs`
- `/home/runner/work/gh-aw/gh-aw/actions/setup/js/bash_command_parser.test.cjs`
- `/home/runner/work/gh-aw/gh-aw/actions/setup/js/fuzz_bash_command_parser_harness.cjs`
- `/home/runner/work/gh-aw/gh-aw/actions/setup/js/copilot_sdk_driver.cjs`
