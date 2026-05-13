# Contributing to DCS-Gate

Thank you for your interest in contributing to this research project. This document describes how to participate constructively.

## Code of Conduct

All contributors are expected to follow the [Code of Conduct](CODE_OF_CONDUCT.md). In short: be respectful, inclusive, and professional.

## Reporting Issues

### Bugs

If you find a bug:

1. **Search existing issues** to avoid duplicates.
2. **Provide a minimal reproducible example** — the exact steps, inputs, and expected vs. actual output.
3. **Include environment details**: Go version, OS, Ollama version, judge model used.
4. **Attach relevant logs** from `/metrics` or HTTP responses.

### Security Issues

**Do not open a public issue for security vulnerabilities.** See [SECURITY.md](SECURITY.md) for responsible disclosure.

## Submitting Pull Requests

### Before you start

- Fork the repository.
- Create a branch: `git checkout -b feature/your-feature-name`
- Keep your branch up to date with `main`.

### Code style & quality

- **Format:** Run `gofmt ./...` before committing.
- **Lint:** Prefer `golangci-lint` if available.
- **Tests:** Add tests for any new functionality. Aim for >80% coverage.
- **Documentation:** Update README, ARCHITECTURE.md, or relevant docs if behavior changes.

### Testing

Run the test suite locally:

```bash
go test -v ./...
go test -race ./...  # check for race conditions
```

For streaming tests, ensure Ollama is running:

```bash
ollama serve &
go test -v -run Stream ./...
```

### Commit messages

- Start with a verb: `Add`, `Fix`, `Refactor`, `Docs`, etc.
- Be concise but descriptive: `Fix judge_thinking extraction for qwen3 thinking mode`
- Reference issues if applicable: `Fixes #42`

### PR checklist

- [ ] Tests pass locally (`go test ./...`)
- [ ] Code formatted (`gofmt`)
- [ ] New tests added for new behavior
- [ ] Documentation updated if behavior changes
- [ ] Commit messages are clear
- [ ] No unrelated changes in this PR

## High-priority areas for contribution

1. **Alternative embedders:** Currently hardcoded to `mxbai-embed-large`. Adding support for other 1024d models (e.g., `nomic-embed-text`) would benefit deployments with different resource constraints.

2. **Additional judge models:** Test suite against `deepseek-r1`, `qwen2.5:32b`, `gpt-oss` variants. Current validation is on `qwen3:14b` thinking mode.

3. **Corpus expansion:** If you have annotated LLM exchanges (question-response pairs) with labels for control patterns or authenticity, we welcome contributions to the triple baseline (`data/baseline_core.jsonl`, etc.). See [`work/dcs-gate/CAMBIOS_V8.md`](work/dcs-gate/CAMBIOS_V8.md#archivos-nuevos-añadir-a-tu-proyecto) for corpus structure.

4. **Cross-language support:** Extend formal markers and intent prototypes to Spanish, French, German, or Mandarin. The methodology is language-agnostic; the tooling is not.

5. **Performance optimization:** Profile the analyzer pipeline and reduce latency for resource-constrained deployments (e.g., edge devices).

6. **Frontend enhancements:** Improve the `/stream-demo` UI or build a standalone web client.

7. **Documentation:** Clarify mathematical formulations, add usage examples, create videos.

## Development workflow

1. **Local setup:**
   ```bash
   git clone https://github.com/Corekeeper-research/dcs-gate
   cd dcs-gate/work/dcs-gate
   go mod tidy
   go build -o dcs-gate .
   ```

2. **Run locally:**
   ```bash
   ollama pull mxbai-embed-large qwen3:14b
   ollama serve &
   ./dcs-gate
   ```

3. **Test your changes:**
   ```bash
   go test -v ./...
   curl -X POST http://localhost:8081/auth \
     -H 'Content-Type: application/json' \
     -d '{"question":"Test?","response":"Test response.","mode":"analyze"}'
   ```

## Questions?

- Open an issue with the `question` label.
- Email: corekeepper@gmail.com
- Check existing documentation in [`docs/`](../docs/) and [`work/dcs-gate/`](../work/dcs-gate/).

---

**Thank you for contributing to better LLM authenticity detection.** 🚀
